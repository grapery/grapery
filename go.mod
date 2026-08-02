module github.com/grapery/grapery

go 1.25.0

require (
	connectrpc.com/connect v1.14.0
	github.com/alibabacloud-go/darabonba-openapi/v2 v2.1.12
	github.com/alibabacloud-go/imageaudit-20191230/v3 v3.0.5
	github.com/alibabacloud-go/tea v1.3.12
	github.com/alibabacloud-go/tea-utils/v2 v2.0.7
	github.com/aliyun/alibaba-cloud-sdk-go v1.63.107
	github.com/aliyun/aliyun-oss-go-sdk v3.0.2+incompatible
	github.com/aliyun/credentials-go v1.4.7
	github.com/apache/rocketmq-clients/golang/v5 v5.1.1-rc1
	github.com/coze-dev/coze-go v0.0.0-20250808084651-adfc6e986262
	github.com/gin-contrib/cors v1.7.6
	github.com/gin-contrib/sessions v1.0.2
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/go-sql-driver/mysql v1.9.0
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/google/uuid v1.6.0
	github.com/grapery/common-protoc/gen v0.0.0-20251109083419-a64eb0113365
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.3.0
	github.com/hibiken/asynq v0.25.1
	github.com/lestrrat-go/jwx/v2 v2.1.6
	github.com/modelcontextprotocol/go-sdk v1.5.0
	github.com/olivere/elastic v6.2.37+incompatible
	github.com/openai/openai-go v1.12.0
	github.com/pkg/errors v0.9.1
	github.com/redis/go-redis/v9 v9.14.1
	github.com/sirupsen/logrus v1.9.3
	github.com/ulule/limiter/v3 v3.11.2
	github.com/volcengine/volcengine-go-sdk v1.1.46
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.43.0
	golang.org/x/net v0.46.0
	golang.org/x/oauth2 v0.35.0
	google.golang.org/genai v1.22.0
	google.golang.org/grpc v1.78.0-dev
	gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df
	gorm.io/datatypes v1.2.6
	gorm.io/driver/mysql v1.5.7
	gorm.io/gorm v1.30.0
)

replace github.com/bufbuild/connect-go => connectrpc.com/connect v1.14.0

require (
	github.com/alibabacloud-go/alibabacloud-gateway-spi v0.0.5 // indirect
	github.com/alibabacloud-go/debug v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.1.2 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/clbanning/mxj/v2 v2.7.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/jsonschema-go v0.4.2 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/lestrrat-go/blackmagic v1.0.3 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/httprc v1.0.6 // indirect
	github.com/lestrrat-go/iter v1.0.2 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/opentracing/opentracing-go v1.2.1-0.20220228012449-10b1cf09e00b // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	github.com/volcengine/volc-sdk-golang v1.0.224 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	google.golang.org/api v0.248.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

require (
	cloud.google.com/go v0.121.6 // indirect
	cloud.google.com/go/auth v0.16.5 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	contrib.go.opencensus.io/exporter/ocagent v0.7.0 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/bytedance/sonic v1.13.3 // indirect
	github.com/bytedance/sonic/loader v0.2.4 // indirect
	github.com/census-instrumentation/opencensus-proto v0.4.1 // indirect
	github.com/cloudwego/base64x v0.1.5 // indirect
	github.com/dchest/siphash v1.2.3 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/gabriel-vasile/mimetype v1.4.9 // indirect
	github.com/gin-contrib/sse v1.1.0 // indirect
	github.com/gin-gonic/gin v1.10.1
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.26.0
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/gomodule/redigo v1.9.2 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.6 // indirect
	github.com/googleapis/gax-go/v2 v2.15.0 // indirect
	github.com/gorilla/context v1.1.2 // indirect
	github.com/gorilla/securecookie v1.1.2 // indirect
	github.com/gorilla/sessions v1.4.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.3 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible // indirect
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/onsi/gomega v1.37.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/snowdreamtech/redistore v0.0.0-20231007100540-6364ca2c97b4 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.3.0 // indirect
	github.com/valyala/fastrand v1.1.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.63.0 // indirect
	go.opentelemetry.io/otel v1.38.0 // indirect
	go.opentelemetry.io/otel/metric v1.38.0 // indirect
	go.opentelemetry.io/otel/trace v1.38.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/arch v0.18.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	golang.org/x/time v0.14.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251103181224-f26f9409b101 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251103181224-f26f9409b101 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
