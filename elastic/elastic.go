package elastic

import (
	"context"
	"time"

	"github.com/olivere/elastic"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type ESDoc interface {
	Index() string
	UpdateIndex() string
	Type() string
	ESID() string
	Template() (string, error)
	LastUsedTime() uint32
	SetLastUsedTime(uint32)
}

var client *elastic.Client

func GetClient() *elastic.Client {
	return client
}

const responseMaxReadLimit = 100000

type Logger struct {
}

func (*Logger) Printf(format string, v ...interface{}) {
	log.Infof(format, v)
}

type ErrorLogger struct {
}

func (*ErrorLogger) Printf(format string, v ...interface{}) {
	log.Errorf(format, v)
}

func Init(address []string) {
	var err error
	client, err = elastic.NewClient(
		elastic.SetURL(address...),
		elastic.SetHealthcheckInterval(30*time.Second),
		elastic.SetErrorLog(&ErrorLogger{}),
		elastic.SetInfoLog(&Logger{}),
		elastic.SetTraceLog(&Logger{}),
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

func GetMultiDocByIds(ctx context.Context, esDoc ESDoc, IDList []string) (results map[string]ESDoc, err error) {
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
	results = make(map[string]ESDoc, len(resp.Docs))
	for _, doc := range resp.Docs {
		if doc.Source == nil {
			continue
		}
		switch esDoc.(type) {
		// case *model.Courier, model.Courier:
		// 	var ret = new(model.Courier)
		// 	err = json.Unmarshal(*doc.Source, ret)
		// 	results[ret.ESID()] = ret

		default:
			log.Errorf("unknown es index type ", esDoc.Index())
		}

	}
	return results, nil
}
