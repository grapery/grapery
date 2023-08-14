module github.com/grapery/grapery

go 1.16

require (
	github.com/bytedance/sonic v1.10.0 // indirect
	github.com/fortytw2/leaktest v1.3.0 // indirect
	github.com/gin-contrib/sessions v0.0.5
	github.com/gin-gonic/gin v1.9.1 // indirect
	github.com/go-playground/validator/v10 v10.15.0 // indirect
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/go-sql-driver/mysql v1.7.0
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/golang/protobuf v1.5.2
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/uuid v1.3.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.11.3
	github.com/klauspost/cpuid/v2 v2.2.5 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/mozillazg/go-httpheader v0.4.0 // indirect
	github.com/olivere/elastic v6.2.37+incompatible
	github.com/onsi/gomega v1.16.0 // indirect
	github.com/pelletier/go-toml/v2 v2.0.9 // indirect
	github.com/pkg/errors v0.9.1
	github.com/rogpeppe/go-internal v1.8.0 // indirect
	github.com/sashabaranov/go-openai v1.8.0
	github.com/sirupsen/logrus v1.9.0
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common v1.0.611
	github.com/tencentyun/cos-go-sdk-v5 v0.7.42
	github.com/ugorji/go v1.2.7 // indirect
	go.uber.org/zap v1.19.1
	golang.org/x/arch v0.4.0 // indirect
	golang.org/x/crypto v0.12.0
	golang.org/x/net v0.14.0 // indirect
	google.golang.org/genproto v0.0.0-20230110181048-76db0878b65f
	google.golang.org/grpc v1.54.0
	google.golang.org/protobuf v1.31.0
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df
	gorm.io/driver/mysql v1.5.1 // indirect
	gorm.io/gorm v1.25.2 // indirect
)

replace (
	github.com/tencentcloud/tencentcloud-sdk-go => /Users/grapestree/go/src/github.com/grapery/grapery/utils/tencentcloud/tencentcloud-sdk-go
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common => /Users/grapestree/go/src/github.com/grapery/grapery/utils/tencentcloud/tencentcloud-sdk-go/tencentcloud/common
)
