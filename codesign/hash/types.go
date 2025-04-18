package hash

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
)

var (
	TypeInvalid         = RegisterHashType(0x0, Metadata{Priority: -1, Name: "INVALID"})
	TypeSHA1            = RegisterHashType(0x1, Metadata{Priority: 1, Name: "SHA1", Size: sha1.Size, DigestSize: sha1.Size, New: sha1.New})
	TypeSHA256          = RegisterHashType(0x2, Metadata{Priority: 3, Name: "SHA256", Size: sha256.Size, DigestSize: sha256.Size, New: sha256.New})
	TypeSHA256Truncated = RegisterHashType(0x3, Metadata{Priority: 2, Name: "SHA256_TRUNCATED", Size: 20, DigestSize: sha256.Size, New: sha256.New})
	TypeSHA384          = RegisterHashType(0x4, Metadata{Priority: 4, Name: "SHA384", Size: sha512.Size384, DigestSize: sha512.Size384, New: sha512.New384})
	TypeSHA512          = RegisterHashType(0x5, Metadata{Priority: 5, Name: "SHA512", Size: sha512.Size, DigestSize: sha512.Size, New: sha512.New})
)
