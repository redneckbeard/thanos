package shims

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/redneckbeard/thanos/stdlib"
)

// URI wraps *url.URL to provide Ruby-compatible accessor methods.
type URI struct {
	u *url.URL
}

// URIParse wraps url.Parse, ignoring errors like Ruby's URI.parse.
func URIParse(s string) *URI {
	u, _ := url.Parse(s)
	if u == nil {
		u = &url.URL{}
	}
	return &URI{u: u}
}

func (u *URI) Scheme() string   { return u.u.Scheme }
func (u *URI) Host() string     { return u.u.Hostname() }
func (u *URI) Hostname() string { return u.u.Hostname() }
func (u *URI) Path() string     { return u.u.Path }
func (u *URI) Query() string    { return u.u.RawQuery }
func (u *URI) Fragment() string { return u.u.Fragment }
func (u *URI) String() string   { return u.u.String() }

// Port returns the port as an integer, or 0 if not present.
func (u *URI) Port() int {
	p := u.u.Port()
	if p == "" {
		return 0
	}
	n, _ := strconv.Atoi(p)
	return n
}

// URIEncodeWwwForm encodes a hash as application/x-www-form-urlencoded.
// Takes an OrderedMap[string, string] and returns the encoded string.
func URIEncodeWwwForm(m *stdlib.OrderedMap[string, string]) string {
	vals := url.Values{}
	for k, v := range m.All() {
		vals.Set(k, v)
	}
	return vals.Encode()
}

// URIDecodeWwwForm decodes a query string into key-value pairs.
func URIDecodeWwwForm(s string) [][]string {
	vals, _ := url.ParseQuery(s)
	var result [][]string
	for k, vs := range vals {
		for _, v := range vs {
			result = append(result, []string{k, v})
		}
	}
	return result
}

// URIJoin joins URI strings by resolving each relative to the previous.
func URIJoin(parts ...string) *URI {
	if len(parts) == 0 {
		return &URI{u: &url.URL{}}
	}
	base, _ := url.Parse(parts[0])
	if base == nil {
		base = &url.URL{}
	}
	for _, p := range parts[1:] {
		ref, _ := url.Parse(p)
		if ref != nil {
			base = base.ResolveReference(ref)
		}
	}
	return &URI{u: base}
}

// Implement fmt.Stringer for puts support.
func (u *URI) GoString() string {
	return u.u.String()
}

// StringForPuts allows fmt.Println to output the URI string directly.
func (u *URI) Format(f interface{ WriteString(string) (int, error) }, _ rune) {
	f.WriteString(u.u.String())
}

// MarshalText makes URI print correctly with fmt.Println.
func (u *URI) MarshalText() ([]byte, error) {
	return []byte(u.u.String()), nil
}

// ensure URI has a String method compatible with fmt.Stringer
var _ interface{ String() string } = (*URI)(nil)

// URIBuildQuery builds a query string from parts.
func URIBuildQuery(parts [][]string) string {
	vals := url.Values{}
	for _, pair := range parts {
		if len(pair) == 2 {
			vals.Add(pair[0], pair[1])
		}
	}
	return vals.Encode()
}

// Small helper: URIEncodeWwwFormFromPairs handles Ruby's array-of-arrays form
func URIEncodeWwwFormFromPairs(pairs [][]string) string {
	var parts []string
	for _, pair := range pairs {
		if len(pair) == 2 {
			parts = append(parts, url.QueryEscape(pair[0])+"="+url.QueryEscape(pair[1]))
		}
	}
	return strings.Join(parts, "&")
}
