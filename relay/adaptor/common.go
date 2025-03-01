package adaptor

import (
	"encoding/json" // 添加此行
	"errors"
	"fmt"
	"bufio"
	"strings"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/client"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/model"
	"bytes"
	"io"
	"net/http"
	"context"
	"strconv"
)

func ErrorWrapper(err error, code string, statusCode int) *model.ErrorWithStatusCode {
	logger.Error(context.TODO(), fmt.Sprintf("[%s]%+v", code, err))

	Error := model.Error{
		Message: err.Error(),
		Type:    "one_api_error",
		Code:    code,
	}
	return &model.ErrorWithStatusCode{
		Error:      Error,
		StatusCode: statusCode,
	}
}


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


	newBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal modified request body failed: %w", err)
	}

	req, err := http.NewRequest(c.Request.Method, fullRequestURL, bytes.NewReader(newBody))
	if err != nil {
		return nil, fmt.Errorf("new request failed: %w", err)
	}

	if err != nil {
		return nil, fmt.Errorf("new request failed: %w", err)
	}
	err = a.SetupRequestHeader(c, req, meta)


	if err != nil {
		return nil, fmt.Errorf("setup request header failed: %w", err)
	}
	// 调用97行的DoRequest
	resp, err := DoRequest(meta.ActualModelName, c, req)
	if err != nil {
		return nil, fmt.Errorf("do request failed: %w", err)
	}

	fmt.Printf("请求结束了+lobe的正常返回包也被修改为openai官方一样的了\n")
	
	return resp, nil
}

func DoRequest(modelnameN string, c *gin.Context, req *http.Request) (*http.Response, error) {
	// 先读取并暂存原始请求体内容
	var requestBody map[string]interface{}

    bodyBytes, err := io.ReadAll(req.Body)
    if err != nil {
        return nil, err
    }

    req.Body.Close() 

    // 将原始JSON解析为map对象，以便修改其中的 stream 参数
    if len(bodyBytes) > 0 {
        if err := json.Unmarshal(bodyBytes, &requestBody); err != nil {
            return nil, err
        }
    } else {
        requestBody = make(map[string]interface{})
    }

    // 强行设置 "stream" 为 false (无论之前是否存在，都覆盖为false)
    requestBody["stream"] = false

    // 序列化回新的 JSON 请求体数据:
	newRequestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
	    return nil,err 
	}

	req.Body=io.NopCloser(bytes.NewReader(newRequestBodyBytes))
	req.ContentLength=int64(len(newRequestBodyBytes))

	// 更新header中的Content-Length字段：
	req.Header.Set("Content-Length", strconv.Itoa(len(newRequestBodyBytes)))

	resp,err:=client.HTTPClient.Do(req)

	if(err!=nil){
	   return nil , err 
	}

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
	} else{
		/* responseBody现在长这样。
		打算吧返回内容修改为以下格式供其他平台调用{"id":"","object":"","created":0,"model":"","choices":[{"delta":{"role":"assistant","content":"xxxxx"},"index":0,"finish_reason":null}]}
		id: chatcmpl-B67KhVjCRchp0jKv0gHB7FbL0hLIX
		event: text
		data: "Hello! How can I assist you today?"

		id: chatcmpl-B67KhVjCRchp0jKv0gHB7FbL0hLIX
		event: stop
		data: "stop"
		// */
		responseBody, err := io.ReadAll(resp.Body)
        fmt.Printf("正常lobe的responseBodyresponseBodyresponseBodyresponseBodyresponseBodyresponseBody: %s\n", responseBody)
        if err != nil {
            return nil, fmt.Errorf("read_response_body_failed: %v", err)
        }
        err = resp.Body.Close()
        if err != nil {
            return nil, fmt.Errorf("close_response_body_failed: %v", err)
        }
        
        fmt.Printf("准备将responseBody更新为choices版本...\n")
        /* responseBody现在长这样。
        打算吧返回内容修改为以下格式供其他平台调用{"id":"","object":"","created":0,"model":"","choices":[{"delta":{"role":"assistant","content":"xxxxx"},"index":0,"finish_reason":null}]}
        id: chatcmpl-B67KhVjCRchp0jKv0gHB7FbL0hLIX
        event: text
        data: "Hello! How can I assist you today?"
    
        id: chatcmpl-B67KhVjCRchp0jKv0gHB7FbL0hLIX
        event: stop
        data: "stop"
        */
    
        // 提取第一个非 "stop" 的 data 值
        var extractedData string
        scanner := bufio.NewScanner(bytes.NewReader(responseBody))
        for scanner.Scan() {
            line := scanner.Text()
            if strings.HasPrefix(line, "data:") {
                // 去除前缀及两边的空白字符
                content := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
                // 移除两侧的双引号
                content = strings.Trim(content, "\"")
                if content != "stop" {
                    extractedData = content
                    break
                }
            }
        }
    
        // 构造目标 JSON 结构，并将 extractedData 嵌入到 message.content 的位置
        type Message struct {
            Role    string  `json:"role"`
            Content string  `json:"content"`
            Refusal *string `json:"refusal"`
        }
        type Choice struct {
            Index        int      `json:"index"`
            Message      Message  `json:"message"`
            Logprobs     *any     `json:"logprobs"` // 可设置为 nil
            FinishReason string   `json:"finish_reason"`
        }
        type PromptTokensDetails struct {
            CachedTokens int `json:"cached_tokens"`
            AudioTokens  int `json:"audio_tokens"`
        }
        type CompletionTokensDetails struct {
            ReasoningTokens            int `json:"reasoning_tokens"`
            AudioTokens                int `json:"audio_tokens"`
            AcceptedPredictionTokens   int `json:"accepted_prediction_tokens"`
            RejectedPredictionTokens   int `json:"rejected_prediction_tokens"`
        }
        type Usage struct {
            PromptTokens            int                     `json:"prompt_tokens"`
            CompletionTokens        int                     `json:"completion_tokens"`
            TotalTokens             int                     `json:"total_tokens"`
            PromptTokensDetails     PromptTokensDetails     `json:"prompt_tokens_details"`
            CompletionTokensDetails CompletionTokensDetails `json:"completion_tokens_details"`
        }
        type Result struct {
            Id                string   `json:"id"`
            Object            string   `json:"object"`
            Created           int64    `json:"created"`
            Model             string   `json:"model"`
            Choices           []Choice `json:"choices"`
            Usage             Usage    `json:"usage"`
            // ServiceTier       string   `json:"service_tier"`
            SystemFingerprint string   `json:"system_fingerprint"`
        }
        res := Result{
            Id:      "chatcmpl-B6BBfld6yNw7QXm9tR0xLvDdkBx2q",
            Object:  "chat.completion",
            Created: 1740812671,
            Model:   modelnameN,
            Choices: []Choice{
                {
                    Index: 0,
                    Message: Message{
                        Role:    "assistant",
                        Content: extractedData, // 第一个 data 的内容
                        Refusal: nil,
                    },
                    Logprobs:     nil,
                    FinishReason: "stop",
                },
            },
            Usage: Usage{
                PromptTokens:     1,
                CompletionTokens: 1,
                TotalTokens:      1,
                PromptTokensDetails: PromptTokensDetails{
                    CachedTokens: 0,
                    AudioTokens:  0,
                },
                CompletionTokensDetails: CompletionTokensDetails{
                    ReasoningTokens:          0,
                    AudioTokens:              0,
                    AcceptedPredictionTokens: 0,
                    RejectedPredictionTokens: 0,
                },
            },
            // ServiceTier:       "default",
            SystemFingerprint: "fp_06737a9306",
        }
    
        finalJSON, err := json.Marshal(res)
        if err != nil {
            // 若 marshal 失败，可作相应处理
            fmt.Printf("marshal result failed: %v\n", err)
        } else {
			fmt.Printf("机器人的回答extractedDataextractedDataextractedData: %s\n", extractedData)


            fmt.Printf("构造完choice后的Final JSON: %s\n", finalJSON)

			
        }

        resp.Body = io.NopCloser(bytes.NewBuffer(finalJSON))
		resp.Header.Set("Content-Type","application/json;charset=utf-8")
		resp.Header.Set("Content-Length",strconv.Itoa(len(finalJSON)))

	}
	_ = req.Body.Close()
	_ = c.Request.Body.Close()
	return resp, nil
}
