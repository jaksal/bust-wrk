package main

import (
	"fmt"
	"net"
	"net/http"
)

// RedirectError specific error type that happens on redirection
type RedirectError struct {
	msg string
}

func (re *RedirectError) Error() string {
	return re.msg
}

// NewRedirectError ...
func NewRedirectError(message string) *RedirectError {
	rt := RedirectError{msg: message}
	return &rt
}

// ByteSize a helper struct that implements the String() method and returns a human readable result. Very useful for %v formatting.
type ByteSize struct {
	Size float64
}

func (bs ByteSize) String() string {
	var rt float64
	var suffix string
	const (
		Byte  = 1
		KByte = Byte * 1024
		MByte = KByte * 1024
		GByte = MByte * 1024
	)

	if bs.Size > GByte {
		rt = bs.Size / GByte
		suffix = "GB"
	} else if bs.Size > MByte {
		rt = bs.Size / MByte
		suffix = "MB"
	} else if bs.Size > KByte {
		rt = bs.Size / KByte
		suffix = "KB"
	} else {
		rt = bs.Size
		suffix = "bytes"
	}

	srt := fmt.Sprintf("%.2f%v", rt, suffix)

	return srt
}

//EstimateHTTPHeadersSize had to create this because headers size was not counted
func EstimateHTTPHeadersSize(headers http.Header) (result int64) {
	result = 0

	for k, v := range headers {
		result += int64(len(k) + len(": \r\n"))
		for _, s := range v {
			result += int64(len(s))
		}
	}

	result += int64(len("\r\n"))

	return result
}

// GetMyIP ...
func GetMyIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
