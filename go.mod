module github.com/grapery/grapery

go 1.16

require (
	github.com/fortytw2/leaktest v1.3.0 // indirect
	github.com/gin-contrib/cors v1.4.0
	github.com/gin-contrib/sessions v0.0.5
	github.com/gin-gonic/gin v1.8.1
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/uuid v1.3.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.5.0
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/mozillazg/go-httpheader v0.3.1 // indirect
	github.com/olivere/elastic v6.2.37+incompatible
	github.com/onsi/gomega v1.16.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/sashabaranov/go-openai v1.5.7
	github.com/sirupsen/logrus v1.8.1
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common v1.0.611
	github.com/tencentyun/cos-go-sdk-v5 v0.7.41
	go.uber.org/zap v1.19.1
	google.golang.org/genproto v0.0.0-20210903162649-d08c68adba83
	google.golang.org/grpc v1.40.0
	google.golang.org/protobuf v1.28.0
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df
	gorm.io/driver/mysql v1.1.2
	gorm.io/gorm v1.21.14
)

replace (
	github.com/tencentcloud/tencentcloud-sdk-go => /Users/grapestree/go/src/github.com/grapery/grapery/utils/tencentcloud/tencentcloud-sdk-go
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common => /Users/grapestree/go/src/github.com/grapery/grapery/utils/tencentcloud/tencentcloud-sdk-go/tencentcloud/common
)
