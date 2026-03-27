package main

import (
	"context"
	"log"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

func main() {
	BASE_URL, hasBaseUrl := os.LookupEnv("ANTHROPIC_BASEURL")
	API_KEY, hasApiKey := os.LookupEnv("ANTHROPIC_KEY")

	if !hasBaseUrl || !hasApiKey {
		log.Printf("ANTHROPIC_BASEURL 或者 ANTHROPIC_KEY 环境变量缺失")
		return
	}

	agentReqClient := anthropic.NewClient(
		option.WithBaseURL(BASE_URL),
		option.WithAPIKey(API_KEY),
	)
	messages := []anthropic.MessageParam{}

	for {
		// 1.发送请求,并接收响应
		resp, err := agentReqClient.Messages.New(context.Background(), anthropic.MessageNewParams{
			MaxTokens: 16 * 1024,
			Model:     anthropic.ModelClaudeSonnet4_6,
			System: []anthropic.TextBlockParam{
				{
					Type: "text",
					Text: "你是一个CLI编程助手，你擅长编写各种语言的代码.",
				},
			},
			Messages: messages,
			Tools:    getTodoListTools(),
		})
		if err != nil {
			panic(err)
		}

		// 2.响应的StopReason如果不是工具调用就退出循环。
		if resp.StopReason != anthropic.StopReasonToolUse {
			return
		}
		// 3.如果是工具调用就执行工具调用,继续循环
		for _, cnt := range resp.Content {
			if cnt.Type == "tool_use" {
				toolExecRes := execTool(cnt.Name, cnt.Input)
				messages = append(messages,
					anthropic.NewAssistantMessage(anthropic.ContentBlockParamUnion{
						OfToolUse: &anthropic.ToolUseBlockParam{
							ID:    cnt.ID,
							Name:  cnt.Name,
							Input: cnt.Input,
						},
					}),
				)
				messages = append(messages,
					anthropic.NewUserMessage(anthropic.ContentBlockParamUnion{
						OfToolResult: &anthropic.ToolResultBlockParam{
							ToolUseID: cnt.ID,
							Content: []anthropic.ToolResultBlockParamContentUnion{
								{OfText: &anthropic.TextBlockParam{
									Text: toolExecRes,
								}},
							},
						},
					}),
				)
			}
		}
	}
}

func getTodoListTools() []anthropic.ToolUnionParam {
	tools := []anthropic.ToolUnionParam{}

	return tools
}

func execTool(toolName string, params []byte) string {

	return ""
}
