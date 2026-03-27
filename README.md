## go-mini-claude-code 使用go练习编写Claude Code，来了解Agent设计

#### 参考 https://github.com/shareAI-lab/mini-claude-code ,可以认为是这个python项目的go语言版本。


#### v0.1 实现了执行bash命令并循环直到任务完成
#### v0.2 实现任务列表
#### v0.3 实现子代理
#### v0.4 实现不支持function_call的大模型的fallback调用


### Agent的执行过程
- 1.发送请求,并接收响应
- 2.响应的StopReason如果不是工具调用就退出循环。
- 3.如果是工具调用就执行工具调用,继续循环


### Anthropic的工具调用的官方文档: 
- https://platform.claude.com/docs/en/agents-and-tools/tool-use/programmatic-tool-calling
- https://platform.claude.com/docs/en/api/go/messages/create
- https://platform.claude.com/docs/en/agents-and-tools/tool-use/tool-reference

