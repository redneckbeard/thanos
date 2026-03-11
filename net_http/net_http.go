package net_http

import (
	"net/http"
	"net/url"
	"strings"
)

// Get performs an HTTP GET and returns just the body string.
// Matches Ruby's Net::HTTP.get(uri) or Net::HTTP.get(host, path).
func Get(rawURL string) string {
	resp, err := http.Get(rawURL)
	if err != nil {
		return ""
	}
	r := newResponse(resp)
	return r.body
}

// GetHostPath performs an HTTP GET with host + path strings.
// Matches Ruby's Net::HTTP.get("example.com", "/path").
func GetHostPath(host, path string) string {
	return Get("http://" + host + path)
}

// GetResponse performs an HTTP GET and returns a Response.
// Matches Ruby's Net::HTTP.get_response(uri).
func GetResponse(rawURL string) *Response {
	resp, err := http.Get(rawURL)
	if err != nil {
		return errResponse(err)
	}
	return newResponse(resp)
}

// GetResponseHostPath performs an HTTP GET with host + path strings.
func GetResponseHostPath(host, path string) *Response {
	return GetResponse("http://" + host + path)
}

// Post performs an HTTP POST with a string body.
// Matches Ruby's Net::HTTP.post(uri, data).
func Post(rawURL, data string) *Response {
	resp, err := http.Post(rawURL, "application/x-www-form-urlencoded", strings.NewReader(data))
	if err != nil {
		return errResponse(err)
	}
	return newResponse(resp)
}

// PostForm performs an HTTP POST with form-encoded parameters.
// Matches Ruby's Net::HTTP.post_form(uri, params).
// params is a map of string keys to string values (single-valued).
func PostForm(rawURL string, params map[string]string) *Response {
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	resp, err := http.PostForm(rawURL, values)
	if err != nil {
		return errResponse(err)
	}
	return newResponse(resp)
}
