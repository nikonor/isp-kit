package httpcli

import (
	"net/http"
)

type GlobalRequestConfig struct {
	BaseUrl   string
	BasicAuth *BasicAuth
	Cookies   []*http.Cookie
	Headers   map[string]string
}

func (c GlobalRequestConfig) configure(req *RequestBuilder) {
	req.baseUrl = c.BaseUrl
	req.basicAuth = c.BasicAuth
	req.cookies = append(req.cookies, c.Cookies...)
	for name, value := range c.Headers {
		req.Header(name, value)
	}
}
