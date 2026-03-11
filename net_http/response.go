package net_http

import (
	"io"
	"net/http"
	"strconv"
)

// Response wraps an HTTP response with Ruby-like semantics.
// body is read eagerly and the underlying body is closed.
type Response struct {
	statusCode int
	body       string
	message    string
	headers    map[string]string
}

// newResponse reads and closes the Go response body, producing a Response.
func newResponse(resp *http.Response) *Response {
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	headers := make(map[string]string, len(resp.Header))
	for k := range resp.Header {
		headers[k] = resp.Header.Get(k)
	}
	return &Response{
		statusCode: resp.StatusCode,
		body:       string(body),
		message:    resp.Status,
		headers:    headers,
	}
}

// errResponse creates a Response for error cases.
func errResponse(err error) *Response {
	return &Response{statusCode: 0, body: "", message: err.Error(), headers: map[string]string{}}
}

// Body returns the response body as a string.
func (r *Response) Body() string {
	return r.body
}

// Code returns the HTTP status code as a string (Ruby convention).
func (r *Response) Code() string {
	return strconv.Itoa(r.statusCode)
}

// CodeInt returns the HTTP status code as an int.
func (r *Response) CodeInt() int {
	return r.statusCode
}

// Message returns the status line (e.g., "200 OK").
func (r *Response) Message() string {
	return r.message
}

// GetHeader returns a response header value (for Ruby's response["Content-Type"]).
func (r *Response) GetHeader(key string) string {
	return r.headers[key]
}
