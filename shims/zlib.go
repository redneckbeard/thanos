package shims

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"io"
)

// ZlibDeflate compresses a string using zlib/deflate.
func ZlibDeflate(s string) string {
	var buf bytes.Buffer
	w, _ := flate.NewWriter(&buf, flate.DefaultCompression)
	w.Write([]byte(s))
	w.Close()
	return buf.String()
}

// ZlibInflate decompresses a deflate-compressed string.
func ZlibInflate(s string) string {
	r := flate.NewReader(bytes.NewReader([]byte(s)))
	defer r.Close()
	out, _ := io.ReadAll(r)
	return string(out)
}

// ZlibGzip compresses a string using gzip.
func ZlibGzip(s string) string {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write([]byte(s))
	w.Close()
	return buf.String()
}

// ZlibGunzip decompresses a gzip-compressed string.
func ZlibGunzip(s string) string {
	r, err := gzip.NewReader(bytes.NewReader([]byte(s)))
	if err != nil {
		return ""
	}
	defer r.Close()
	out, _ := io.ReadAll(r)
	return string(out)
}
