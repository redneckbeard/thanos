package net_http

import (
	"fmt"
	"net/http"
	"strings"
)

// Client wraps an HTTP connection to a host, matching Ruby's Net::HTTP instance.
type Client struct {
	host   string
	port   int
	useSSL bool
	client *http.Client
}

// NewClient creates a Client. Port 0 means use default (443 for SSL, 80 otherwise).
func NewClient(host string, port int, useSSL bool) *Client {
	if port == 0 {
		if useSSL {
			port = 443
		} else {
			port = 80
		}
	}
	return &Client{host: host, port: port, useSSL: useSSL, client: &http.Client{}}
}

func (c *Client) baseURL() string {
	scheme := "http"
	if c.useSSL {
		scheme = "https"
	}
	defaultPort := 80
	if c.useSSL {
		defaultPort = 443
	}
	if c.port == defaultPort {
		return fmt.Sprintf("%s://%s", scheme, c.host)
	}
	return fmt.Sprintf("%s://%s:%d", scheme, c.host, c.port)
}

func (c *Client) url(path string) string {
	return c.baseURL() + path
}

// Get performs an HTTP GET request.
func (c *Client) Get(path string) *Response {
	resp, err := c.client.Get(c.url(path))
	if err != nil {
		return errResponse(err)
	}
	return newResponse(resp)
}

// Post performs an HTTP POST request with a string body.
func (c *Client) Post(path, data string) *Response {
	resp, err := c.client.Post(c.url(path), "application/x-www-form-urlencoded", strings.NewReader(data))
	if err != nil {
		return errResponse(err)
	}
	return newResponse(resp)
}

// Put performs an HTTP PUT request.
func (c *Client) Put(path, data string) *Response {
	return c.doRequest("PUT", path, data)
}

// Patch performs an HTTP PATCH request.
func (c *Client) Patch(path, data string) *Response {
	return c.doRequest("PATCH", path, data)
}

// Delete performs an HTTP DELETE request.
func (c *Client) Delete(path string) *Response {
	return c.doRequest("DELETE", path, "")
}

// Head performs an HTTP HEAD request.
func (c *Client) Head(path string) *Response {
	resp, err := c.client.Head(c.url(path))
	if err != nil {
		return errResponse(err)
	}
	return newResponse(resp)
}

// DoRequest sends a custom Request and returns the Response.
func (c *Client) DoRequest(req *Request) *Response {
	httpReq, err := req.build(c.url(req.path))
	if err != nil {
		return errResponse(err)
	}
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return errResponse(err)
	}
	return newResponse(resp)
}

func (c *Client) doRequest(method, path, data string) *Response {
	var body *strings.Reader
	if data != "" {
		body = strings.NewReader(data)
	} else {
		body = strings.NewReader("")
	}
	req, err := http.NewRequest(method, c.url(path), body)
	if err != nil {
		return errResponse(err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return errResponse(err)
	}
	return newResponse(resp)
}
