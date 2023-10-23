module github.com/grapery/grapery

go 1.21.0

require (
	github.com/gin-contrib/sessions v0.0.5
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/go-sql-driver/mysql v1.7.0
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/google/uuid v1.3.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.11.3
	github.com/olivere/elastic v6.2.37+incompatible
	github.com/pkg/errors v0.9.1
	github.com/sashabaranov/go-openai v1.8.0
	github.com/sirupsen/logrus v1.9.0
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common v1.0.611
	github.com/tencentyun/cos-go-sdk-v5 v0.7.42
	go.uber.org/zap v1.19.1
	golang.org/x/crypto v0.12.0
	google.golang.org/genproto v0.0.0-20230110181048-76db0878b65f
	google.golang.org/grpc v1.54.0
	google.golang.org/protobuf v1.31.0
	gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df
	gorm.io/driver/mysql v1.5.1
	gorm.io/gorm v1.25.2
)

require (
	github.com/boj/redistore v0.0.0-20180917114910-cd5dcc76aeff // indirect
	github.com/bytedance/sonic v1.10.0 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20230717121745-296ad89f973d // indirect
	github.com/chenzhuoyu/iasm v0.9.0 // indirect
	github.com/clbanning/mxj v1.8.4 // indirect
	github.com/fortytw2/leaktest v1.3.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.2 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/gin-gonic/gin v1.9.1 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.15.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/gomodule/redigo v2.0.0+incompatible // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/gorilla/context v1.1.1 // indirect
	github.com/gorilla/securecookie v1.1.1 // indirect
	github.com/gorilla/sessions v1.2.1 // indirect
	github.com/grapery/common-protoc v0.0.0-20231023141933-d146b53de518 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.2.5 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mozillazg/go-httpheader v0.4.0 // indirect
	github.com/onsi/ginkgo v1.16.4 // indirect
	github.com/onsi/gomega v1.16.0 // indirect
	github.com/pelletier/go-toml/v2 v2.0.9 // indirect
	github.com/rogpeppe/go-internal v1.8.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.11 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/arch v0.4.0 // indirect
	golang.org/x/net v0.14.0 // indirect
	golang.org/x/sys v0.11.0 // indirect
	golang.org/x/text v0.12.0 // indirect
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/tencentcloud/tencentcloud-sdk-go => /Users/grapestree/go/src/github.com/grapery/grapery/utils/tencentcloud/tencentcloud-sdk-go
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common => /Users/grapestree/go/src/github.com/grapery/grapery/utils/tencentcloud/tencentcloud-sdk-go/tencentcloud/common
)
