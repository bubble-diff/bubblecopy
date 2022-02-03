package main

import (
	"bytes"
)

var httpMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "DELETE": true, "HEAD": true,
	"TRACE": true, "OPTIONS": true, "PATCH": true,
}

// guessProtocol 根据TCP数据判断其协议类型
func guessProtocol(payload []byte) (protocol string) {
	if isHttpRequestData(payload) {
		return HttpType
	}
	return UnknownType
}

// isHttpRequestData 判断是否符合HTTP/1.x
func isHttpRequestData(payload []byte) bool {
	// see https://stackoverflow.com/questions/25047905/http-request-minimum-size-in-bytes
	if len(payload) < 26 {
		return false
	}
	idx := bytes.IndexByte(payload, byte(' '))
	if idx < 0 {
		return false
	}
	method := string(payload[:idx])
	if ok := httpMethods[method]; !ok {
		return false
	}
	return true
}
