package aliyun

import (
	"net/url"
	"strings"
)

// OwnsHTTPSMediaURL permits only this application's configured OSS origin.
// No arbitrary URLs, ports, credentials, fragments, redirects or proxy query
// parameters enter the publication vision audit through a user request.
func (c *Client) OwnsHTTPSMediaURL(raw string) bool {
	if c == nil || c.config == nil {
		return false
	}
	endpoint := strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(c.config.Endpoint), "https://"), "http://")
	endpoint = strings.TrimSuffix(endpoint, "/")
	if !strings.HasSuffix(endpoint, ".aliyuncs.com") || strings.ContainsAny(endpoint, "/@:") {
		return false
	}
	return ownedMediaURL(raw, c.config.Bucket+"."+endpoint)
}
func ownedMediaURL(raw, host string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host == host && u.User == nil && u.RawQuery == "" && u.Fragment == "" && u.Path != "" && u.Path != "/" && !strings.Contains(u.Path, "..")
}
