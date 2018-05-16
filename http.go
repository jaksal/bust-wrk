package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

var defaultTransport http.Transport

func init() {
	// Customize the Transport to have larger connection pool
	defaultRoundTripper := http.DefaultTransport
	defaultTransportPointer, ok := defaultRoundTripper.(*http.Transport)
	if !ok {
		panic(fmt.Sprintf("defaultRoundTripper not an *http.Transport"))
	}
	defaultTransport = *defaultTransportPointer // dereference it to get a copy of the struct that the pointer points to
	defaultTransport.MaxIdleConns = 1000
	defaultTransport.MaxIdleConnsPerHost = 1000
}

func newHTTPClient() *http.Client {
	return &http.Client{
		Transport: &defaultTransport,
	}
}

func doRequest(client *http.Client, path string, method string, headerList map[string]string, urlParamList map[string]string, bodyList map[string]string) (map[string]interface{}, time.Duration, int, error) {
	// fmt.Printf("param path=%s method=%s header=%+v url_param=%+v body=%+v\n", path, method, headerList, urlParamList, bodyList)
	var buf io.Reader
	if bodyList != nil {
		data := url.Values{}
		for key, val := range bodyList {
			data.Add(key, val)
		}
		buf = bytes.NewBufferString(data.Encode())
	}

	if urlParamList != nil {
		q := url.Values{}
		for k, v := range urlParamList {
			q.Add(k, v)
		}
		path += "?" + q.Encode()
	}

	req, err := http.NewRequest(method, path, buf)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("An error occured http new request %s", err)
	}
	if headerList != nil {
		for key, val := range headerList {
			req.Header.Add(key, val)
		}
	}

	if method == "POST" {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, 0, err
	}
	duration := time.Since(start)

	if resp == nil {
		return nil, 0, 0, fmt.Errorf("empty response")
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("An error occured reading body %s", err)
	}

	var parsed map[string]interface{}

	var respSize int
	if resp.StatusCode == http.StatusOK {
		respSize = len(body) + int(EstimateHTTPHeadersSize(resp.Header))
		if len(body) > 0 {
			if err := json.Unmarshal(body, &parsed); err != nil {
				return nil, 0, 0, fmt.Errorf("json Unmarshal error %s body=%s", err, string(body))
			}
		}
	} else if resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusTemporaryRedirect {
		respSize = int(resp.ContentLength) + int(EstimateHTTPHeadersSize(resp.Header))
	} else {
		// fmt.Println("received status code", resp.StatusCode, "from", resp.Header, "content", string(body), req)
		return nil, 0, 0, fmt.Errorf("resp status code err=%d body=%s", resp.StatusCode, string(body))
	}
	return parsed, duration, respSize, nil
}
