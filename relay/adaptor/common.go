package adaptor

import (
	"encoding/json" // 添加此行
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/client"
	"bytes"
	"github.com/songquanpeng/one-api/relay/meta"
	"io"
	"net/http"
)

func SetupCommonRequestHeader(c *gin.Context, req *http.Request, meta *meta.Meta) {
	req.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
	req.Header.Set("Accept", c.Request.Header.Get("Accept"))
	if meta.IsStream && c.Request.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "text/event-stream")
	}
}

func DoRequestHelper(a Adaptor, c *gin.Context, meta *meta.Meta, requestBody io.Reader) (*http.Response, error) {
	fullRequestURL, err := a.GetRequestURL(meta)

	// fmt.Printf("最终请求的完整url-common//////fullRequestURLfullRequestURLfullRequestURLfullRequestURLfullRequestURLfullRequestURLfullRequestURL: %s\n", fullRequestURL)
	if err != nil {
		return nil, fmt.Errorf("get request url failed: %w", err)
	}

	bodyBytes, err := io.ReadAll(requestBody)
	if err != nil {
		return nil, fmt.Errorf("read request body failed: %w", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal request body failed: %w", err)
	}

	// 添加 "stream": false
	payload["stream"] = false

	newBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal modified request body failed: %w", err)
	}

	req, err := http.NewRequest(c.Request.Method, fullRequestURL, bytes.NewReader(newBody))
	if err != nil {
		return nil, fmt.Errorf("new request failed: %w", err)
	}
	// req, err := http.NewRequest(c.Request.Method, fullRequestURL, requestBody)

	// bodyBytes, err := io.ReadAll(requestBody)
	// fmt.Printf("最终body-Request bodyRequest bodyRequest bodyRequest bodyRequest bodyRequest body: %s\n", string(bodyBytes))

	if err != nil {
		return nil, fmt.Errorf("new request failed: %w", err)
	}
	err = a.SetupRequestHeader(c, req, meta)


	if err != nil {
		return nil, fmt.Errorf("setup request header failed: %w", err)
	}
	resp, err := DoRequest(c, req)
	if err != nil {
		return nil, fmt.Errorf("do request failed: %w", err)
	}
	return resp, nil
}

func DoRequest(c *gin.Context, req *http.Request) (*http.Response, error) {
	resp, err := client.HTTPClient.Do(req)
	fmt.Printf("最终请求最终请求最终请求最终请求最终请求最终请求最终请求最终请求最终请求最终请求最终请求最终请求最终请求最终请求最终请求最终请求最终请求最终请求\n")
	fmt.Printf("Request URL: %s\n", req.URL.String())
    fmt.Printf("Request Headers: %v\n", req.Header)
	// bodyBytes, err := io.ReadAll(req.Body)
	// fmt.Printf("Request Body: %s\n", string(bodyBytes))

	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.New("resp is nil")
	}
	_ = req.Body.Close()
	_ = c.Request.Body.Close()
	return resp, nil
}
