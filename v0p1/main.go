package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

func main() {
	ANTHROPIC_BASEURL := os.Getenv("ANTHROPIC_BASEURL")
	ANTHROPIC_KEY := os.Getenv("ANTHROPIC_KEY")

	if ANTHROPIC_BASEURL == "" || ANTHROPIC_KEY == "" {
		log.Println("请使用 export ANTHROPIC_BASEURL=\"xxxx\" 或者export ANTHROPIC_KEY=\"yyy\" 设置环境变量")
	}

	if len(os.Args) < 2 {
		fmt.Println("请提供参数: ./gmcc \"帮我分析一下这个项目实现的功能\"")
		os.Exit(1)
	}

	userAskQuestion := os.Args[1]

	client := anthropic.NewClient(
		option.WithBaseURL(ANTHROPIC_BASEURL),
		option.WithAPIKey(ANTHROPIC_KEY),
	)

	messsages := []anthropic.MessageParam{
		{
			Role: anthropic.MessageParamRoleUser,
			Content: []anthropic.ContentBlockParamUnion{
				anthropic.NewTextBlock(userAskQuestion),
			},
		},
	}

	loopCnt := 1
	// 如果返回工具调用就将工具调用的结果拼接到messages中，持续提交给LLM接口。如果返回的是非工具调用就退出执行。
	for {
		log.Printf("第 %d 次交互", loopCnt)
		loopCnt++

		msg, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
			MaxTokens: 16 * 1024,
			Model:     "kimi-for-coding",
			System: []anthropic.TextBlockParam{
				{Text: "你是一个CLI工具助手。"},
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

		messsages = append(messsages, anthropic.MessageParam{
			Role:    anthropic.MessageParamRoleAssistant,
			Content: toContentBlockParams(msg.Content),
		})

		if msg.StopReason != anthropic.StopReasonToolUse {
			// 非工具调用
			for _, msgCnt := range msg.Content {
				log.Println("非工具调用 ", msgCnt.Text)
			}
			return
		}

		// 工具调用
		for _, msgCnt := range msg.Content {
			switch msgCnt.Type {
			case "tool_use":
				toolDataBytes, err := msgCnt.Input.MarshalJSON()
				if err != nil {
					panic(err)
				}
				toolCallInfo := map[string]string{}
				err = json.Unmarshal(toolDataBytes, &toolCallInfo)
				if err != nil {
					panic(err)
				}
				log.Println("工具调用 msgCnt.Type: ", "tool_use; ", "tool_name: ", msgCnt.Name, "msgCnt.input.command", toolCallInfo["command"])
				execOutput := RunBashCommand(toolCallInfo["command"])

				messsages = append(messsages, anthropic.MessageParam{
					Role: anthropic.MessageParamRoleUser,
					Content: []anthropic.ContentBlockParamUnion{
						{OfToolResult: &anthropic.ToolResultBlockParam{
							ToolUseID: msgCnt.ID,
							Content: []anthropic.ToolResultBlockParamContentUnion{
								{OfText: &anthropic.TextBlockParam{
									Type: "text",
									Text: execOutput,
								}},
							},
						}},
					},
				})

			default:
				log.Println("工具调用 msgCnt.Type", msgCnt.Type, "msgCnt.Text", msgCnt.Text)
			}
		}
	}

}

func RunBashCommand(command string) string {
	cmd := exec.CommandContext(context.Background(), "sh", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err.Error()
	}

	return string(output)
}

func toContentBlockParams(blocks []anthropic.ContentBlockUnion) []anthropic.ContentBlockParamUnion {
	result := make([]anthropic.ContentBlockParamUnion, 0, len(blocks))

	for _, block := range blocks {
		switch block.Type {
		case "text":
			result = append(result, anthropic.ContentBlockParamUnion{
				OfText: &anthropic.TextBlockParam{
					Text: block.Text,
				},
			})

		case "tool_use":
			result = append(result, anthropic.ContentBlockParamUnion{
				OfToolUse: &anthropic.ToolUseBlockParam{
					Type:  "tool_use",
					ID:    block.ID,
					Name:  block.Name,
					Input: block.Input, // 已经是 json.RawMessage
				},
			})

		case "thinking":
			result = append(result, anthropic.ContentBlockParamUnion{
				OfThinking: &anthropic.ThinkingBlockParam{
					Thinking:  block.Thinking,
					Signature: block.Signature,
				},
			})

		case "redacted_thinking":
			result = append(result, anthropic.ContentBlockParamUnion{
				OfRedactedThinking: &anthropic.RedactedThinkingBlockParam{
					Data: block.Data,
				},
			})
		}
	}

	return result
}
