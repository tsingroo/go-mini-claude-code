package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"

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

	loopCount := 1

	for {
		log.Printf("-------------   第 %d 次交互 -------------\n", loopCount)

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
			log.Println("完成任务! 正常退出。")
			return
		}
		// 3.如果是工具调用就执行工具调用,继续循环
		for _, cnt := range resp.Content {
			if cnt.Type == "tool_use" {
				toolExecRes, toolExecErr := execTool(cnt.Name, cnt.Input)
				hasErr := false
				if toolExecErr != nil {
					hasErr = true
					toolExecRes = toolExecErr.Error()
				}
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
							IsError:   anthropic.Bool(hasErr),
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
	tools := []anthropic.ToolUnionParam{
		// Bash
		{
			OfTool: &anthropic.ToolParam{
				Name:        "Bash",
				Description: anthropic.String("MacOs或者Linux上的Shell或者Bash终端"),
				InputSchema: anthropic.ToolInputSchemaParam{
					Type: "object",
					Properties: map[string]any{
						"command": map[string]any{
							"type":        "string",
							"description": "要执行的shell命令",
						},
					},
					Required: []string{"command"},
				},
			},
		},
		// Task List
		{
			OfTool: &anthropic.ToolParam{
				Name:        "TaskList",
				Description: anthropic.String("用于规划和记录复杂任务和多步骤任务的工具"),
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
									"description": "任务列表项的描述",
								},
							},
						},
					},
					Required: []string{""},
				},
			},
		},
	}

	return tools
}

func execTool(toolName string, params []byte) (string, error) {
	if toolName == "Bash" {
		command := string(params)
		cmd := exec.CommandContext(context.Background(), "sh", "-c", command)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", err
		}

		return string(output), nil
	}
	// Task List
	taskList := []TaskListItem{}
	if err := json.Unmarshal(params, &taskList); err != nil {
		log.Printf("模型返回的任务列表格式错误, %s", string(params))
		return "", errors.New("反序列化错误, 任务列表格式错误")
	}
	taskExecRes := ""
	completedCnt := 0
	taskCnt := len(taskList)
	for _, taskItem := range taskList {
		taskStatusIcon := []string{"", "[x]", "[>]", "[ ]"}
		if taskItem.Status > 3 || taskItem.Status < 1 {
			continue
		}
		if taskItem.Status == 3 {
			completedCnt++
		}
		taskExecRes += taskStatusIcon[taskItem.Status] + " " + taskItem.Desc + "\n"
	}

	summary := fmt.Sprintf(" completed %d of %d items ", completedCnt, taskCnt)
	log.Println("任务进度更新： " + summary)
	taskExecRes += summary

	return taskExecRes, nil
}

type TaskListItem struct {
	Status int    `json:"status"`
	Desc   string `json:"desc"`
}
