package aliyun

import (
	"context"
	"reflect"
	"testing"
)

func TestWanxiangClient_GenerateVideoFromImage(t *testing.T) {
	type fields struct {
		APIKey   string
		Endpoint string
	}
	type args struct {
		ctx    context.Context
		imgURL string
		prompt string
		params *T2IParams
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *TaskResponse
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "test",
			fields: fields{
				APIKey:   "your-api-key",
				Endpoint: "your-endpoint",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &WanxiangClient{
				APIKey:   tt.fields.APIKey,
				Endpoint: tt.fields.Endpoint,
			}
			got, err := c.GenerateVideoFromImage(tt.args.ctx, tt.args.imgURL, tt.args.prompt, tt.args.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("WanxiangClient.GenerateVideoFromImage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WanxiangClient.GenerateVideoFromImage() = %v, want %v", got, tt.want)
			}
		})
	}
}
