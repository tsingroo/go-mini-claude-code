package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

func main() {
	ANTHROPIC_BASEURL := os.Getenv("ANTHROPIC_BASEURL")
	ANTHROPIC_KEY := os.Getenv("ANTHROPIC_KEY")

	client := anthropic.NewClient(
		option.WithBaseURL(ANTHROPIC_BASEURL),
		option.WithAPIKey(ANTHROPIC_KEY),
	)

	messsages := []anthropic.MessageParam{
		{
			Role: anthropic.MessageParamRoleUser,
			Content: []anthropic.ContentBlockParamUnion{
				anthropic.NewTextBlock("当前目录的 ./v0p1/main.go 实现了什么功能?"),
			},
		},
	}

	msg, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
		MaxTokens: 16 * 1024,
		Model:     "kimi-for-coding",
		System: []anthropic.TextBlockParam{
			{Text: "你是一个CLI工具助手。你可以使用Bash命令来完成任务。"},
		},
		Messages: messsages,
		Tools: []anthropic.ToolUnionParam{
			{
				OfTool: &anthropic.ToolParam{
					Name:        "bash",
					Description: anthropic.String("linux 或者 MacOS 系统上的bash命令行"),
					InputSchema: anthropic.ToolInputSchemaParam{
						Type: "object",
						Properties: map[string]any{
							"command": map[string]any{"type": "string"},
						},
						Required: []string{"command"},
					},
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}
	if msg.StopReason == anthropic.StopReasonToolUse {
		// 产生工具调用
		for _, msgCnt := range msg.Content {
			switch msgCnt.Type {
			case "tool_use":
				toolDataBytes, err := msgCnt.Input.MarshalJSON()
				if err != nil {
					panic(err)
				}
				toolCallInfo := map[string]any{}
				err = json.Unmarshal(toolDataBytes, &toolCallInfo)
				if err != nil {
					panic(err)
				}
				log.Println("工具调用 msgCnt.Type: ", "tool_use; ", "tool_name: ", msgCnt.Name, "msgCnt.input.command", toolCallInfo["command"])
			default:
				log.Println("工具调用 msgCnt.Type", msgCnt.Type, "msgCnt.Text", msgCnt.Text)
			}
		}
	} else {
		// 非工具调用
		for _, msgCnt := range msg.Content {
			log.Println("非工具调用 msgCnt.Type: ", msgCnt.Type, "msgCnt.Text", msgCnt.Text)
		}
	}

}
