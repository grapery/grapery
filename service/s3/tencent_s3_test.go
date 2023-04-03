package s3

import (
	"context"
	"reflect"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestS3Client_ListBuckets(t *testing.T) {
	type fields struct {
		TencentS3Client *cos.Client
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s3c := &S3Client{
				TencentS3Client: tt.fields.TencentS3Client,
			}
			got, err := s3c.ListBuckets(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("S3Client.ListBuckets() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("S3Client.ListBuckets() = %v, want %v", got, tt.want)
			}
		})
	}
}
