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

	loopCount := 1
	for {
		log.Printf("-------------   第 %d 次交互 -------------\n", loopCount)
		loopCount++

		// 1.发送请求，并接受响应
		resp, err := agentClient.Messages.New(ctx, anthropic.MessageNewParams{
			MaxTokens: 16 * 1024,
			System: []anthropic.TextBlockParam{
				{
					Text: "你是一个CLI命令行编码助手。无论任务难易程度如何，你都需要制定一个多步骤的任务计划来完成用户的请求。任务计划中的任务项如果完全不依赖主代理的上下文就使用子代理去执行这个任务。",
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
			log.Printf("主代理输入Token数 %.1f k", float64(resp.Usage.InputTokens)/1000.0)
			log.Printf("主代理输出Token数 %.1f k", float64(resp.Usage.OutputTokens)/1000.0)
			log.Println("调用结束，主代理退出")
			return
		}
		// 3.否则调用工具，继续循环
		for _, cnt := range resp.Content {
			if cnt.Type != "tool_use" {
				log.Printf("工具调用中的内容输出: %s \n", cnt.Text)
				continue
			}

			log.Printf("调用工具: %s , %s", cnt.Name, string(cnt.Input))
			toolExecRes, toolExecErr := executeTools(cnt.Name, cnt.Input, modelInfo)
			hasErr := false
			if toolExecErr != nil {
				hasErr = true
				toolExecRes = toolExecErr.Error()
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
					IsError:   anthropic.Bool(hasErr),
					Content: []anthropic.ToolResultBlockParamContentUnion{
						{
							OfText: &anthropic.TextBlockParam{Text: toolExecRes},
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
		log.Println("---------- 开始调用子代理执行任务 ------------")
		return utils.ExecSubagentTool(modelInfo, toolParams)
	}

	return "", errors.New("不存在的工具")
}
