package utils

import "github.com/anthropics/anthropic-sdk-go"

// 如果主代理可以使用子代理工具，如果是子代理就不能继续使用子代理工具
func GetTools(isSubAgent bool) []anthropic.ToolUnionParam {
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
