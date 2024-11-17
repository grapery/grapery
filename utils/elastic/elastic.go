package elastic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/olivere/elastic"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/grapery/grapery/utils/log"
)

type ElasticDoc interface {
	Index() string
	Type() string
	ElasticID() string
	LastUsedTime() int64
	SetLastUsedTime(int64)
}

var (
	client          *elastic.Client
	IsEnableElastic bool
)

func GetClient() *elastic.Client {
	return client
}

type Logger struct {
	logger *zap.Logger
}

func (l *Logger) Printf(format string, v ...interface{}) {
	l.logger.Info(fmt.Sprintf(format, v...))
}

type ErrorLogger struct {
	logger *zap.Logger
}

func (l *ErrorLogger) Printf(format string, v ...interface{}) {
	l.logger.Error(fmt.Sprintf(format, v...))
}

func Init(address []string) {
	var err error
	client, err = elastic.NewClient(
		elastic.SetURL(address...),
		elastic.SetHealthcheckInterval(60*time.Second),
		elastic.SetErrorLog(&ErrorLogger{
			logger: log.Log(),
		}),
		elastic.SetInfoLog(&Logger{
			logger: log.Log(),
		}),
		elastic.SetTraceLog(&Logger{
			logger: log.Log(),
		}),
		elastic.SetSniff(false),
	)
	if err != nil {
		panic(err)
	}
	defer client.Stop()
	c := context.Background()
	for i := 0; i < len(address); i++ {
		_, _, err := client.Ping(address[i]).Do(c)
		if err != nil {
			panic(errors.WithMessage(err, "Address at "+address[i]))
		}
	}
}

func GetMultiDocByIds(ctx context.Context, esDoc ElasticDoc, IDList []string) (results map[string]ElasticDoc, err error) {
	multiReq := GetClient().MultiGet()
	for idx := range IDList {
		multiReq.Add(elastic.NewMultiGetItem().Index(esDoc.Index()).Type(esDoc.Type()).Id(IDList[idx]))
	}
	resp, err := multiReq.Do(ctx)
	if err != nil {
		return nil, errors.WithMessagef(err, "id[%v]", IDList)
	}
	if len(resp.Docs) == 0 {
		return nil, nil
	}
	results = make(map[string]ElasticDoc, len(resp.Docs))
	for _, doc := range resp.Docs {
		if doc.Source == nil {
			continue
		}
		switch esDoc.(type) {
		case *ElasticUser:
			var ret = new(ElasticUser)
			err = json.Unmarshal(*doc.Source, ret)
			if err != nil {
				return nil, err
			}
			results[ret.ElasticID()] = ret
		case *ElasticGroup:
			var ret = new(ElasticGroup)
			err = json.Unmarshal(*doc.Source, ret)
			if err != nil {
				return nil, err
			}
			results[ret.ElasticID()] = ret
		case *ElasticProject:
			var ret = new(ElasticProject)
			err = json.Unmarshal(*doc.Source, ret)
			if err != nil {
				return nil, err
			}
			results[ret.ElasticID()] = ret
		case *ElasticItem:
			var ret = new(ElasticItem)
			err = json.Unmarshal(*doc.Source, ret)
			if err != nil {
				return nil, err
			}
			results[ret.ElasticID()] = ret
		default:
			log.Log().Error(fmt.Sprintf("unknown es index type: %+v", esDoc.Index()))
		}

	}
	return results, nil
}
