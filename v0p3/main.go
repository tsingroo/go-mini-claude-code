package main

import (
	"context"
	"go-mini-claude-code/utils"
	"log"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

func main() {
	modelInfo, err := utils.GetModelInfo()
	if err != nil {
		log.Println(err.Error())
		return
	}

	if len(os.Args) < 2 {
		log.Println("请提供参数: ./gmcc \"帮我分析一下这个项目实现的功能\"")
		return
	}

	userAskQuestion := os.Args[1]
	agentClient := anthropic.NewClient(
		option.WithBaseURL(modelInfo.BaseUrl),
		option.WithAPIKey(modelInfo.ApiKey),
	)
	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(userAskQuestion)),
	}
	ctx := context.Background()

	for {
		// 1.发送请求，并接受响应
		resp, err := agentClient.Messages.New(ctx, anthropic.MessageNewParams{
			MaxTokens: 16 * 1024,
			System: []anthropic.TextBlockParam{
				{
					Text: "你是一个CLI命令行编码助手，你在响应用户请求的时候不要直接修改或分析代码，你需要先制定任务计划。计划中的每一项你都需要使用子代理来执行任务,子代理执行完任务后你可以获取它的响应来决定你的下一步计划。",
				},
			},
			Model:    anthropic.Model(modelInfo.ModelName),
			Messages: messages,
			Tools:    getTools(false),
		})

		if err != nil {
			panic(err)
		}
		// 2,如果StopReason不是ToolUser就退出
		if resp.StopReason != anthropic.StopReasonToolUse {
			for _, cnt := range resp.Content {
				log.Printf("非工具调用内容输出: %s \n", cnt.Text)
			}
			log.Println("调用结束，主代理退出")
			return
		}
		// 3.否则调用工具，继续循环
		for _, cnt := range resp.Content {
			if cnt.Type != "tool_use" {
				log.Printf("工具调用中的内容输出: %s \n", cnt.Text)
				continue
			}
			output, err := executeTools(cnt.Name, cnt.Input)
			if err != nil {
				// TODO: 添加错误
				continue
			}
			messages = append(messages, anthropic.NewAssistantMessage(anthropic.ContentBlockParamUnion{
				OfToolUse: &anthropic.ToolUseBlockParam{
					ID:    cnt.ID,
					Name:  cnt.Name,
					Input: cnt.Input,
				},
			}))
			messages = append(messages, anthropic.NewUserMessage(anthropic.ContentBlockParamUnion{
				OfToolResult: &anthropic.ToolResultBlockParam{
					ToolUseID: cnt.ID,
					IsError:   anthropic.Bool(false),
					Content: []anthropic.ToolResultBlockParamContentUnion{
						{
							OfText: &anthropic.TextBlockParam{Text: output},
						},
					},
				},
			}))
		}
	}
}

// 如果主代理可以使用子代理工具，如果是子代理就不能继续使用子代理工具
func getTools(isSubAgent bool) []anthropic.ToolUnionParam {
	commonTools := []anthropic.ToolUnionParam{
		// 包含 Bash, TaskList 工具
		{
			OfTool: &anthropic.ToolParam{
				Name:        "Bash",
				Description: anthropic.String("Linux或Macos上的Bash命令终端"),
				InputSchema: anthropic.ToolInputSchemaParam{
					Type: "object",
					Properties: map[string]any{
						"command": map[string]any{
							"type":        "string",
							"description": "要在终端中执行的Bash命令",
						},
					},
					Required: []string{"command"},
				},
			},
		},
		{
			OfTool: &anthropic.ToolParam{
				Name:        "TaskList",
				Description: anthropic.String("任务列表和计划列表管理工具"),
				InputSchema: anthropic.ToolInputSchemaParam{
					Type: "object",
					Properties: map[string]any{
						"list": map[string]any{
							"type":        "array",
							"description": "任务列表项",
							"items": map[string]any{
								"status": map[string]any{
									"type":        "number",
									"description": "任务状态: 1表示未执行，2表示进行中，3表示已执行",
								},
								"desc": map[string]any{
									"type":        "string",
									"description": "任务项的描述",
								},
							},
						},
					},
					Required: []string{"list"},
				},
			},
		},
	}
	subgAgentTools := []anthropic.ToolUnionParam{
		{
			OfTool: &anthropic.ToolParam{
				Name:        "SubAgent",
				Description: anthropic.String("子代理: 可以在干净的上下文中执行探索代码或者修改代码的任务"),
				InputSchema: anthropic.ToolInputSchemaParam{
					Type: "object",
					Properties: map[string]any{
						"subAgentType": map[string]any{
							"type":        "string",
							"description": "子代理类型: 'explore'表示探索类型，'code'表示编码类型",
						},
						"prompt": map[string]any{
							"type":        "string",
							"description": "子代理执行任务需要的提示词",
						},
					},
					Required: []string{"subAgentType", "prompt"},
				},
			},
		},
	}

	if !isSubAgent {
		commonTools = append(commonTools, subgAgentTools...)
	}

	return commonTools
}

func executeTools(name string, toolParams []byte) (string, error) {

	return "", nil
}

// Bash调用的响应结构
type BashCommandParam struct {
	Command string `json:"command"`
}

// 任务列表调用的响应结构
type TaskListParam struct {
	List []struct {
		Status int    `json:"status"`
		Desc   string `json:"desc"` // 每一项任务的文本描述
	} `json:"list"`
}

// 子代理调用的响应结构
type SubAgentParam struct {
	SubAgentType string `json:"subAgentType"` // 探索、编码
	Prompt       string `json:"prompt"`       // 给子代理的提示词
}
