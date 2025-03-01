package openai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/conv"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
	"io"
	"fmt"
	"net/http"
	"strings"
)

const (
	dataPrefix       = "data: "
	done             = "[DONE]"
	dataPrefixLength = len(dataPrefix)
)

func StreamHandler(c *gin.Context, resp *http.Response, relayMode int) (*model.ErrorWithStatusCode, string, *model.Usage) {
	responseText := ""
	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := strings.Index(string(data), "\n"); i >= 0 {
			return i + 1, data[0:i], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	})
	dataChan := make(chan string)
	stopChan := make(chan bool)
	var usage *model.Usage
	go func() {
		for scanner.Scan() {
			data := scanner.Text()
			if len(data) < dataPrefixLength { // ignore blank line or wrong format
				continue
			}
			if data[:dataPrefixLength] != dataPrefix && data[:dataPrefixLength] != done {
				continue
			}
			if strings.HasPrefix(data[dataPrefixLength:], done) {
				dataChan <- data
				continue
			}
			switch relayMode {
			case relaymode.ChatCompletions:
				var streamResponse ChatCompletionsStreamResponse
				err := json.Unmarshal([]byte(data[dataPrefixLength:]), &streamResponse)
				if err != nil {
					logger.SysError("error unmarshalling stream response: " + err.Error())
					dataChan <- data // if error happened, pass the data to client
					continue         // just ignore the error
				}
				if len(streamResponse.Choices) == 0 {
					// but for empty choice, we should not pass it to client, this is for azure
					continue // just ignore empty choice
				}
				dataChan <- data
				for _, choice := range streamResponse.Choices {
					responseText += conv.AsString(choice.Delta.Content)
				}
				if streamResponse.Usage != nil {
					usage = streamResponse.Usage
				}
			case relaymode.Completions:
				dataChan <- data
				var streamResponse CompletionsStreamResponse
				err := json.Unmarshal([]byte(data[dataPrefixLength:]), &streamResponse)
				if err != nil {
					logger.SysError("error unmarshalling stream response: " + err.Error())
					continue
				}
				for _, choice := range streamResponse.Choices {
					responseText += choice.Text
				}
			}
		}
		stopChan <- true
	}()
	common.SetEventStreamHeaders(c)
	c.Stream(func(w io.Writer) bool {
		select {
		case data := <-dataChan:
			if strings.HasPrefix(data, "data: [DONE]") {
				data = data[:12]
			}
			// some implementations may add \r at the end of data
			data = strings.TrimSuffix(data, "\r")
			c.Render(-1, common.CustomEvent{Data: data})
			return true
		case <-stopChan:
			return false
		}
	})
	err := resp.Body.Close()
	if err != nil {
		return ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError), "", nil
	}
	return nil, responseText, usage
}

func Handler(c *gin.Context, resp *http.Response, promptTokens int, modelName string) (*model.ErrorWithStatusCode, *model.Usage) {
	var textResponse SlimTextResponse
	responseBody, err := io.ReadAll(resp.Body)
	fmt.Printf("responseBodyresponseBodyresponseBodyresponseBodyresponseBodyresponseBody: %s\n", responseBody)
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

	// 构造目标 JSON 结构，并将 extractedData 嵌入到 "content" 字段
	type Result struct {
		Id      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		Model   string `json:"model"`
		Choices []struct {
			Delta struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"delta"`
			Index        int         `json:"index"`
			FinishReason interface{} `json:"finish_reason"`
		} `json:"choices"`
	}

	res := Result{
		Id:      "",
		Object:  "",
		Created: 0,
		Model:   "",
		Choices: []struct {
			Delta struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"delta"`
			Index        int         `json:"index"`
			FinishReason interface{} `json:"finish_reason"`
		}{
			{
				Delta: struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				}{
					Role:    "assistant",
					Content: extractedData, // 将提取到的内容填入此处
				},
				Index:        0,
				FinishReason: nil,
			},
		},
	}

	finalJSON, err := json.Marshal(res)
	if err != nil {
		// 若 marshal 失败，可作相应处理
		fmt.Printf("marshal result failed: %v\n", err)
	} else {
		fmt.Printf("Final JSON: %s\n", finalJSON)
	}
	





	if err != nil {
		return ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError), nil
	}
	err = resp.Body.Close()
	if err != nil {
		return ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError), nil
	}
	// err = json.Unmarshal(responseBody, &textResponse)
	// if err != nil {
	// 	return ErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError), nil
	// }
	// if textResponse.Error.Type != "" {
	// 	return &model.ErrorWithStatusCode{
	// 		Error:      textResponse.Error,
	// 		StatusCode: resp.StatusCode,
	// 	}, nil
	// }
	// Reset response body
	resp.Body = io.NopCloser(bytes.NewBuffer(responseBody))

	// We shouldn't set the header before we parse the response body, because the parse part may fail.
	// And then we will have to send an error response, but in this case, the header has already been set.
	// So the HTTPClient will be confused by the response.
	// For example, Postman will report error, and we cannot check the response at all.
	// for k, v := range resp.Header {
	// 	c.Writer.Header().Set(k, v[0])
	// }
	// c.Writer.WriteHeader(resp.StatusCode)
	// _, err = io.Copy(c.Writer, resp.Body)
	// if err != nil {
	// 	return ErrorWrapper(err, "copy_response_body_failed", http.StatusInternalServerError), nil
	// }
	// err = resp.Body.Close()
	// if err != nil {
	// 	return ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError), nil
	// }

	// if textResponse.Usage.TotalTokens == 0 {
	// 	completionTokens := 0
	// 	for _, choice := range textResponse.Choices {
	// 		completionTokens += CountTokenText(choice.Message.StringContent(), modelName)
	// 	}
	// 	textResponse.Usage = model.Usage{
	// 		PromptTokens:     promptTokens,
	// 		CompletionTokens: completionTokens,
	// 		TotalTokens:      promptTokens + completionTokens,
	// 	}
	// }

	textResponse.Usage = model.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 1}
	// return nil, &textResponse.Usage
	return nil, &textResponse.Usage
}
