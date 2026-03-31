package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/exec"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

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

// 调用Bash工具
func ExecBashTool(params []byte) (string, error) {
	var bashCommand BashCommandParam
	if err := json.Unmarshal(params, &bashCommand); err != nil {
		errMsg := fmt.Sprintf("Bash工具参数错误: %s", string(params))
		log.Println(errMsg)
		return "", errors.New(errMsg)
	}

	cmd := exec.CommandContext(context.Background(), "sh", "-c", bashCommand.Command)
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return string(outputBytes), nil
}

// 调用任务列表工具
func ExecTaskListTool(params []byte) (string, error) {
	var taskList TaskListParam
	if err := json.Unmarshal(params, &taskList); err != nil {
		errMsg := fmt.Sprintf("TaskList参数错误: %s", string(params))
		log.Println(errMsg)
		return "", errors.New(errMsg)
	}
	taskListCnt := ""
	statusDsp := []string{"", "[ ]", "[>]", "[x]"}
	completedCnt := 0
	for _, listItem := range taskList.List {
		if listItem.Status > 3 || listItem.Status < 1 {
			continue
		}
		if listItem.Status == 3 {
			completedCnt++
		}
		taskListCnt += fmt.Sprintf("%s  %s \n", statusDsp[listItem.Status], listItem.Desc)
	}
	taskListCnt += fmt.Sprintf(" completed %d of %d \n", completedCnt, len(taskList.List))

	return taskListCnt, nil
}

// 调用子代理工具
func ExecSubagentTool(modelInfo *ModelInfo, params []byte) (string, error) {
	var subAgentParam SubAgentParam
	if err := json.Unmarshal(params, &subAgentParam); err != nil {
		errMsg := fmt.Sprintf("子代理工具参数错误: %s", string(params))
		log.Println(errMsg)
		return "", errors.New(errMsg)
	}
	subAgentClient := anthropic.NewClient(
		option.WithBaseURL(modelInfo.BaseUrl),
		option.WithAPIKey(modelInfo.ApiKey),
	)
	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(subAgentParam.Prompt)),
	}

	loopCount := 1
	for {
		log.Printf("-------------   子代理第 %d 次交互 -------------\n", loopCount)
		loopCount++

		// 1.发送请求，接收响应
		resp, err := subAgentClient.Messages.New(context.TODO(), anthropic.MessageNewParams{
			MaxTokens: 16 * 1024,
			Model:     anthropic.Model(modelInfo.ModelName),
			System: []anthropic.TextBlockParam{
				{
					Text: "你是一个主代理生成的子代理，你在全新的上下文中执行任务，所以你返回信息的时候要返回完整必要但不冗余的信息来供主代理做出下一步的决策。",
				},
			},
			Messages: messages,
			Tools:    GetTools(true),
		})
		if err != nil {
			panic(err.Error())
		}
		// 2.如果不是ToolUse就结束调用
		if resp.StopReason != anthropic.StopReasonToolUse {
			retMsg := "子代理执行完毕"
			if len(resp.Content) > 0 {
				retMsg = resp.Content[0].Text
			}

			log.Printf("    > 子代理输入Token数 %.1f k", float64(resp.Usage.InputTokens)/1000.0)
			log.Printf("    > 子代理输出Token数 %.1f k", float64(resp.Usage.OutputTokens)/1000.0)
			log.Printf("    > 子代理返回内容: %s \n", retMsg)

			return retMsg, nil
		}
		// 3.如果是ToolUse就执行工具调用
		for _, cnt := range resp.Content {
			if cnt.Type != "tool_use" {
				continue
			}
			output, err := subagentExecTool(cnt.Name, cnt.Input)
			if err != nil {
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

func subagentExecTool(name string, toolParams []byte) (string, error) {
	// Bash
	if name == "Bash" {
		return ExecBashTool(toolParams)
	}
	// TaskList
	if name == "TaskList" {
		return ExecTaskListTool(toolParams)
	}

	return "", errors.New("不存在的工具")
}
