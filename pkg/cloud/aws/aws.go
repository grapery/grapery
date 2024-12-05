package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type AWS struct {
	session *session.Session
	s3      *s3.S3
}

func NewAWS(region string) *AWS {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))
	return &AWS{
		session: sess,
		s3:      s3.New(sess),
	}
}
