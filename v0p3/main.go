package main

import (
	"context"
	"errors"
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
			Tools:    utils.GetTools(false),
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
			output, err := executeTools(cnt.Name, cnt.Input, modelInfo)
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

func executeTools(name string, toolParams []byte, modelInfo *utils.ModelInfo) (string, error) {
	// Bash
	if name == "Bash" {
		return utils.ExecBashTool(toolParams)
	}
	// TaskList
	if name == "TaskList" {
		return utils.ExecTaskListTool(toolParams)
	}

	// SubAgent
	if name == "SubAgent" {
		return utils.ExecSubagentTool(modelInfo, toolParams)
	}

	return "", errors.New("不存在的工具")
}
