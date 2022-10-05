package rpc

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type GrHttpClient struct {
	client http.Client
}

/**
 * NewHttpClient
 * @Author：Jack-Z
 * @Description:
 * @return *GrHttpClient
 */
func NewHttpClient() *GrHttpClient {
	client := http.Client{
		Timeout: time.Duration(3) * time.Second,
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   5,
			MaxConnsPerHost:       100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	return &GrHttpClient{client: client}
}

/**
 * Get
 * @Author：Jack-Z
 * @Description: http的get请求
 * @receiver c
 * @param url
 * @return []byte
 * @return error
 */
func (c *GrHttpClient) Get(url string) ([]byte, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.responseHandler(request)
}

/**
 * responseHandler
 * @Author：Jack-Z
 * @Description: 响应的处理
 * @receiver c
 * @param request
 * @return []byte
 * @return error
 */
func (c *GrHttpClient) responseHandler(request *http.Request) ([]byte, error) {
	do, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}
	if do.StatusCode != http.StatusOK {
		errinfo := fmt.Sprintf("response status id %d", do.StatusCode)
		return nil, errors.New(errinfo)
	}
	reader := bufio.NewReader(do.Body)
	defer do.Body.Close()
	var buf = make([]byte, 127)
	var body []byte
	for {
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if err == io.EOF || n == 0 {
			break
		}
		body = append(body, buf[:n]...)
		if n < len(buf) {
			break
		}
	}
	return body, nil
}
