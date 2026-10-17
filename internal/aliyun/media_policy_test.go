package aliyun

import "testing"

func TestPublicationMediaOriginPolicy(t *testing.T) {
	c := &Client{config: &Config{Bucket: "app", Endpoint: "oss-cn-shanghai.aliyuncs.com"}}
	for _, raw := range []string{"https://app.oss-cn-shanghai.aliyuncs.com/fragments/a.jpg"} {
		if !c.OwnsHTTPSMediaURL(raw) {
			t.Fatal(raw)
		}
	}
	for _, raw := range []string{"http://app.oss-cn-shanghai.aliyuncs.com/a.jpg", "https://127.0.0.1/a.jpg", "https://localhost/a.jpg", "https://app.oss-cn-shanghai.aliyuncs.com.evil.test/a.jpg", "https://other.oss-cn-shanghai.aliyuncs.com/a.jpg", "https://app.oss-cn-shanghai.aliyuncs.com:443/a.jpg", "https://app.oss-cn-shanghai.aliyuncs.com/a.jpg?url=http://localhost", "https://user@app.oss-cn-shanghai.aliyuncs.com/a.jpg"} {
		if c.OwnsHTTPSMediaURL(raw) {
			t.Fatalf("accepted unsafe %s", raw)
		}
	}
}
