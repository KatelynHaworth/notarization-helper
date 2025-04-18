module github.com/KatelynHaworth/notarization-helper/v2

go 1.22

toolchain go1.23.4

require (
	github.com/aws/aws-sdk-go-v2 v1.36.3
	github.com/aws/aws-sdk-go-v2/service/s3 v1.79.1
	github.com/go-resty/resty/v2 v2.11.0
	github.com/golang-jwt/jwt/v5 v5.2.2
	github.com/kr/pretty v0.1.0
	github.com/rs/zerolog v1.15.0
	golang.org/x/sync v0.1.0
	gopkg.in/yaml.v2 v2.2.8
	howett.net/plist v0.0.0-20181124034731-591f970eefbb
)

require (
	github.com/LiamHaworth/macos-golang v0.3.0 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.7.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.18.15 // indirect
	github.com/aws/smithy-go v1.22.2 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/kr/text v0.1.0 // indirect
	github.com/spf13/cobra v1.9.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	golang.org/x/net v0.17.0 // indirect
)

replace github.com/LiamHaworth/macos-golang => ../macos-golang