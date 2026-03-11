package types

// NetHTTPClass is the class shell for Net::HTTP. Method specs are populated
// by net_http/types.go init().
var NetHTTPClass = NewClass("HTTP", "Object", nil, ClassRegistry)
