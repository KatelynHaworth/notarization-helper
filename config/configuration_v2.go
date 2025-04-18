package config

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/KatelynHaworth/notarization-helper/v2/notarize/api"
	"github.com/go-resty/resty/v2"
	"github.com/golang-jwt/jwt/v5"
)

const (
	// defaultAscExpiryDuration specifies how long
	// an App Store Connect auth token signed by this
	// code should be valid for.
	//
	// The App Store Connect API only allows for a
	// 20-minute expiry unless specific conditions
	// are meet (see the [API Documentation]) so it's
	// easier to default to the maximum allowed for all.
	//
	// [API Documentation]: https://developer.apple.com/documentation/AppStoreConnectAPI/generating-tokens-for-api-requests#Determine-the-Appropriate-Token-Lifetime
	defaultAscExpiryDuration = 20 * time.Minute

	// defaultAspExpiryDuration specifies how long
	// an ASP token obtained from the notary service
	// is to be considered valid for.
	defaultAspExpiryDuration = 5 * time.Minute
)

type ConfigurationV2 struct {
	NotaryAuth *ConfigurationV2_NotaryAuth `json:"notary_auth" yaml:"notary_auth"`
	Packages   []Package                   `json:"packages" yaml:"packages"`
}

func (config *ConfigurationV2) GetPackages() []Package {
	return config.Packages
}

func (_ *ConfigurationV2) _isConfig() {}

type ConfigurationV2_NotaryAuth struct {
	KeyId       string  `json:"key_id" yaml:"key_id"`
	KeyFile     string  `json:"key_file" yaml:"key_file"`
	KeyIssuerId *string `json:"key_issuer_id" yaml:"key_issuer_id"`

	username            string
	appSpecificPassword string
	teamId              string

	tokenLock sync.Mutex
	token     *ConfigurationV2_NotaryAuthToken
}

func (auth *ConfigurationV2_NotaryAuth) GetAuthToken() (*ConfigurationV2_NotaryAuthToken, error) {
	auth.tokenLock.Lock()
	defer auth.tokenLock.Unlock()

	var err error
	switch {
	case !auth.token.hasExpired():
		return auth.token, nil

	case len(auth.username) != 0 && len(auth.appSpecificPassword) != 0:
		if err = auth.retrieveAppSpecificPasswordToken(); err != nil {
			err = fmt.Errorf("obtain token for app specific password: %w", err)
		}

	default:
		if err = auth.signAppStoreConnectToken(); err != nil {
			err = fmt.Errorf("sign app store connect token: %w", err)
		}
	}

	return auth.token, err
}

func (auth *ConfigurationV2_NotaryAuth) AuthenticateApiRequests(req *resty.Request) error {
	token, err := auth.GetAuthToken()
	if err != nil {
		return fmt.Errorf("get auth token: %w", err)
	}

	req.SetAuthScheme("Bearer")
	req.SetAuthToken(token.signedToken)

	if token.isAspToken && len(token.teamId) > 0 {
		req.SetHeader("X-Developer-Team-Id", token.teamId)
	}

	return nil
}

func (auth *ConfigurationV2_NotaryAuth) retrieveAppSpecificPasswordToken() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := api.GetAppSpecificPasswordToken(ctx, func(req *resty.Request) error {
		req.SetBasicAuth(auth.username, auth.appSpecificPassword)
		return nil
	})

	if err != nil {
		return fmt.Errorf("request app specific password token: %w", err)
	}

	token := new(ConfigurationV2_NotaryAuthToken)
	token.Issued = jwt.NewNumericDate(time.Now())
	token.Expiry = jwt.NewNumericDate(token.Issued.Add(defaultAspExpiryDuration))
	token.Audience = "appstoreconnect-v1"
	token.isAspToken = true
	token.teamId = auth.teamId
	token.signedToken = resp.Token

	auth.token = token
	return nil
}

func (auth *ConfigurationV2_NotaryAuth) signAppStoreConnectToken() error {
	key, err := auth.loadAppStoreConnectKey()
	if err != nil {
		return fmt.Errorf("load app store connect key to sign auth token: %w", err)
	}

	token := new(ConfigurationV2_NotaryAuthToken)
	token.Issued = jwt.NewNumericDate(time.Now())
	token.Expiry = jwt.NewNumericDate(token.Issued.Add(defaultAscExpiryDuration))
	token.Audience = "appstoreconnect-v1"

	if auth.KeyIssuerId != nil {
		// Issuer is only set if the key is for
		// a team on App Store Connect than a
		// key from a single user
		token.Issuer = *auth.KeyIssuerId
	} else {
		token.Subject = "user"
	}

	tokenJwt := jwt.NewWithClaims(jwt.SigningMethodES256, token, func(token *jwt.Token) {
		// Set the key ID in the JWT header so that
		// the App Store Connect API knows which key
		// to use for validating the JWT
		token.Header["kid"] = auth.KeyId
	})

	if token.signedToken, err = tokenJwt.SignedString(key); err != nil {
		return fmt.Errorf("sign auth token JWT: %w", err)
	}

	auth.token = token
	return nil
}

func (auth *ConfigurationV2_NotaryAuth) loadAppStoreConnectKey() (*ecdsa.PrivateKey, error) {
	var (
		rawKey []byte
		err    error
	)

	if envName, found := strings.CutPrefix(auth.KeyFile, "ENV:"); found {
		if rawKey, err = base64.StdEncoding.DecodeString(os.Getenv(envName)); err != nil {
			return nil, fmt.Errorf("decode auth key from environment variable: %w", err)
		}
	} else {
		if rawKey, err = os.ReadFile(auth.KeyFile); err != nil {
			return nil, fmt.Errorf("read auth key from file: %w", err)
		}
	}

	key, err := x509.ParsePKCS8PrivateKey(rawKey)
	if err != nil {
		return nil, fmt.Errorf("parse auth key: %w", err)
	} else if ecdsaKey, ok := key.(*ecdsa.PrivateKey); !ok || ecdsaKey.Curve != elliptic.P256() {
		return nil, fmt.Errorf("parsed auth key is not a ES256 private key")
	} else {
		return ecdsaKey, nil
	}
}

type ConfigurationV2_NotaryAuthToken struct {
	Issued   *jwt.NumericDate `json:"iat"`
	Expiry   *jwt.NumericDate `json:"exp"`
	Issuer   string           `json:"iss,omitempty"`
	Subject  string           `json:"sub,omitempty"`
	Audience string           `json:"aud"`

	signedToken string
	isAspToken  bool
	teamId      string
}

func (token *ConfigurationV2_NotaryAuthToken) GetExpirationTime() (*jwt.NumericDate, error) {
	return token.Expiry, nil
}

func (token *ConfigurationV2_NotaryAuthToken) GetIssuedAt() (*jwt.NumericDate, error) {
	return token.Issued, nil
}

func (token *ConfigurationV2_NotaryAuthToken) GetNotBefore() (*jwt.NumericDate, error) {
	return nil, nil
}

func (token *ConfigurationV2_NotaryAuthToken) GetIssuer() (string, error) {
	return token.Issuer, nil
}

func (token *ConfigurationV2_NotaryAuthToken) GetSubject() (string, error) {
	return token.Subject, nil
}

func (token *ConfigurationV2_NotaryAuthToken) GetAudience() (jwt.ClaimStrings, error) {
	return jwt.ClaimStrings{
		token.Audience,
	}, nil
}

func (token *ConfigurationV2_NotaryAuthToken) hasExpired() bool {
	if token == nil || token.Expiry.Before(time.Now()) {
		return true
	}

	return false
}
