package openai

import (
	"github.com/songquanpeng/one-api/common/env"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/adaptor/doubao"
	"github.com/songquanpeng/one-api/relay/adaptor/minimax"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
	"io"
	"net/http"
	"strings"
)

type Adaptor struct {
	ChannelType int
}

func (a *Adaptor) Init(meta *meta.Meta) {
	a.ChannelType = meta.ChannelType
}

func (a *Adaptor) GetRequestURL(meta *meta.Meta) (string, error) {
	switch meta.ChannelType {
	case channeltype.Azure:
		if meta.Mode == relaymode.ImagesGenerations {
			// https://learn.microsoft.com/en-us/azure/ai-services/openai/dall-e-quickstart?tabs=dalle3%2Ccommand-line&pivots=rest-api
			// https://{resource_name}.openai.azure.com/openai/deployments/dall-e-3/images/generations?api-version=2024-03-01-preview
			fullRequestURL := fmt.Sprintf("%s/openai/deployments/%s/images/generations?api-version=%s", meta.BaseURL, meta.ActualModelName, meta.Config.APIVersion)
			return fullRequestURL, nil
		}

		// https://learn.microsoft.com/en-us/azure/cognitive-services/openai/chatgpt-quickstart?pivots=rest-api&tabs=command-line#rest-api
		requestURL := strings.Split(meta.RequestURLPath, "?")[0]
		requestURL = fmt.Sprintf("%s?api-version=%s", requestURL, meta.Config.APIVersion)
		task := strings.TrimPrefix(requestURL, "/v1/")
		model_ := meta.ActualModelName
		model_ = strings.Replace(model_, ".", "", -1)
		//https://github.com/songquanpeng/one-api/issues/1191
		// {your endpoint}/openai/deployments/{your azure_model}/chat/completions?api-version={api_version}
		requestURL = fmt.Sprintf("/openai/deployments/%s/%s", model_, task)
		return GetFullRequestURL(meta.BaseURL, requestURL, meta.ChannelType), nil
	case channeltype.Minimax:
		return minimax.GetRequestURL(meta)
	case channeltype.Doubao:
		return doubao.GetRequestURL(meta)
	default:

		// 带/chat/completions的接口
		// return GetFullRequestURL(meta.BaseURL, meta.RequestURLPath, meta.ChannelType), nil

		// 自定义和openai中需要填写完整的请求URL。如https://xx.xx/webapi/chat/openai, https://xx.xx/api/openai/v1/chat/completions
		return GetFullRequestURL(meta.BaseURL, meta.RequestURLPath, meta.ChannelType), nil



		
		
	}
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Request, meta *meta.Meta) error {
	adaptor.SetupCommonRequestHeader(c, req, meta)
	if meta.ChannelType == channeltype.Azure {
		req.Header.Set("api-key", meta.APIKey)
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+meta.APIKey)
	req.Header.Set("X-Forwarded-For", "127.0.0.1")

	var Xlobechatauth = env.String("X-lobe-chat-auth", "eyJhbGciOiJIUzI1NiJ9.eyJhY2Nlc3NDb2RlIjoiMTIzNDU2IiwidXNlcklkIjoiYzg4Yzg5NTEtYWE5MC00MzI4LTk3NTgtMTZjYTBmMWY1NWRjIiwiaWF0IjoxNzQwNTYyMzM0LCJleHAiOjE3NDA2NDg2MzR9.FiNXolhYVmxeeiXmaISm0DWzsaVnRdPTm7lKQsW8MzM")
	// lobeAccessCode校验jwt

	req.Header.Set("X-lobe-chat-auth", Xlobechatauth)

	fmt.Printf("req.Headerreq.Headerreq.Headerreq.Headerreq.Headerreq.Headerreq.Headerreq.Headerreq.Headerreq.Headerreq.Headerreq.Headerreq.Headerreq.Headerreq.Headerreq.Header")
	fmt.Printf("req.Header: %v\n", req.Header)
	if meta.ChannelType == channeltype.OpenRouter {
		req.Header.Set("HTTP-Referer", "https://github.com/songquanpeng/one-api")
		req.Header.Set("X-Title", "One API")
	}
	return nil
}

func (a *Adaptor) ConvertRequest(c *gin.Context, relayMode int, request *model.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	return request, nil
}

func (a *Adaptor) ConvertImageRequest(request *model.ImageRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	return request, nil
}

func (a *Adaptor) DoRequest(c *gin.Context, meta *meta.Meta, requestBody io.Reader) (*http.Response, error) {
	
	// fmt.Printf("DoRequestDoRequestDoRequestDoRequestDoRequestDoRequestDoRequestDoRequestDoRequestDoRequestDoRequestDoRequestDoRequest")
	// fmt.Printf("meta.BaseURL: %s\n", meta.BaseURL)
	// fmt.Printf("meta.RequestURLPath: %s\n", meta.RequestURLPath)
	// fmt.Printf("meta.ChannelType: %d\n", meta.ChannelType)
	// fmt.Printf("meta.Config.APIVersion: %s\n", meta.Config.APIVersion)
	// fmt.Printf("meta.ActualModelName: %s\n", meta.ActualModelName)
	// fmt.Printf("meta.APIKey: %s\n", meta.APIKey)
	// fmt.Printf("meta.IsStream: %v\n", meta.IsStream)
	// fmt.Printf("meta.Mode: %d\n", meta.Mode)
	// fmt.Printf("meta.PromptTokens: %d\n", meta.PromptTokens)
	return adaptor.DoRequestHelper(a, c, meta, requestBody)


}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, meta *meta.Meta) (usage *model.Usage, err *model.ErrorWithStatusCode) {
	// fmt.Printf("结果DoResponseDoResponseDoResponseDoResponseDoResponseDoResponseDoResponseDoResponseDoResponse")

	// if resp != nil && resp.Body != nil {
    //     if meta.IsStream {
    //         fmt.Println("响应为流式数据，跳过打印返回包内容。")
    //     } else {
    //         bodyBytes, errRead := io.ReadAll(resp.Body)
    //         if errRead != nil {
    //             fmt.Printf("读取响应包失败: %v\n", errRead)
    //         } else {
    //             fmt.Printf("Response Body: %s\n", string(bodyBytes))
    //         }
    //         // 重新设置 resp.Body，供后续的 Handler 使用
    //         resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
    //     }
    // }




	if meta.IsStream {
		var responseText string
		err, responseText, usage = StreamHandler(c, resp, meta.Mode)
		// fmt.Printf("responseTextresponseTextresponseTextresponseTextresponseTextresponseTextresponseTextresponseText: %s\n",responseText)


		if usage == nil || usage.TotalTokens == 0 {
			usage = ResponseText2Usage(responseText, meta.ActualModelName, meta.PromptTokens)

		}
		if usage.TotalTokens != 0 && usage.PromptTokens == 0 { // some channels don't return prompt tokens & completion tokens
			usage.PromptTokens = meta.PromptTokens
			usage.CompletionTokens = usage.TotalTokens - meta.PromptTokens
		}
	} else {
		fmt.Printf("判定为非流式，判定为非流式判定为非流式判定为非流式判定为非流式判定为非流式判定为非流式判定为非流式判定为非流式")
		switch meta.Mode {
		case relaymode.ImagesGenerations:
			err, _ = ImageHandler(c, resp)
		default:
			err, usage = Handler(c, resp, meta.PromptTokens, meta.ActualModelName)
		}
	}
	return
}

func (a *Adaptor) GetModelList() []string {
	_, modelList := GetCompatibleChannelMeta(a.ChannelType)
	return modelList
}

func (a *Adaptor) GetChannelName() string {
	channelName, _ := GetCompatibleChannelMeta(a.ChannelType)
	return channelName
}
