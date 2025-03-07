package openai

import (
	"bufio"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/conv"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
	"io"
	"bytes"
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
    if err != nil {
        return ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError), nil
    }
    err = resp.Body.Close()
    if err != nil {
        return ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError), nil
    }
    err = json.Unmarshal(responseBody, &textResponse)
    if err != nil {
        return ErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError), nil
    }
    if textResponse.Error.Type != "" {
        return &model.ErrorWithStatusCode{
            Error:      textResponse.Error,
            StatusCode: resp.StatusCode,
        }, nil
    }
    // Reset response body
    resp.Body = io.NopCloser(bytes.NewBuffer(responseBody))

    // We shouldn't set the header before we parse the response body, because the parse part may fail.
    // And then we will have to send an error response, but in this case, the header has already been set.
    // So the HTTPClient will be confused by the response.
    // For example, Postman will report error, and we cannot check the response at all.

	// fmt.Printf("----------响应给客户端的数据: %s\n", string(responseBody))

    for k, v := range resp.Header {
        c.Writer.Header().Set(k, v[0])
    }
    c.Writer.WriteHeader(resp.StatusCode)
	// 响应给客户端
    _, err = io.Copy(c.Writer, resp.Body)

    if err != nil {
        return ErrorWrapper(err, "copy_response_body_failed", http.StatusInternalServerError), nil
    }
    err = resp.Body.Close()
    if err != nil {
        return ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError), nil
    }

    if textResponse.Usage.TotalTokens == 0 || (textResponse.Usage.PromptTokens == 0 && textResponse.Usage.CompletionTokens == 0) {
        completionTokens := 0
        for _, choice := range textResponse.Choices {
            completionTokens += CountTokenText(choice.Message.StringContent(), modelName)
        }
        textResponse.Usage = model.Usage{
            PromptTokens:     promptTokens,
            CompletionTokens: completionTokens,
            TotalTokens:      promptTokens + completionTokens,
        }
    }
    return nil, &textResponse.Usage
}