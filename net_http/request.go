package net_http

import (
	"net/http"
	"strings"
)

// Request represents an HTTP request with headers, matching Ruby's Net::HTTP::Get etc.
type Request struct {
	method  string
	path    string
	body    string
	headers map[string]string
}

// NewRequest creates a Request with the given HTTP method and path.
func NewRequest(method, path string) *Request {
	return &Request{
		method:  method,
		path:    path,
		headers: map[string]string{},
	}
}

// GetHeader returns a request header value.
func (r *Request) GetHeader(key string) string {
	return r.headers[key]
}

// SetHeader sets a request header.
func (r *Request) SetHeader(key, val string) {
	r.headers[key] = val
}

// SetBody sets the request body.
func (r *Request) SetBody(body string) {
	r.body = body
}

// build creates an *http.Request from this Request.
func (r *Request) build(url string) (*http.Request, error) {
	var bodyReader *strings.Reader
	if r.body != "" {
		bodyReader = strings.NewReader(r.body)
	} else {
		bodyReader = strings.NewReader("")
	}
	req, err := http.NewRequest(r.method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	for k, v := range r.headers {
		req.Header.Set(k, v)
	}
	return req, nil
}
