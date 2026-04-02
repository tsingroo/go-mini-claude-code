## go-mini-claude-code 使用go练习编写Claude Code，来了解Agent设计

#### 参考 https://github.com/shareAI-lab/mini-claude-code ,可以认为是这个python项目的go语言版本。

### 已实现
- v0.1 实现了执行bash命令并循环直到任务完成
- v0.2 实现任务列表
- v0.3 实现子代理(SubAgent)，加入模型环境变量

### 未实现
- v0.4 Skills实现是在调用这个技能时，拼接了一段提示词到会话中。这里先不实现了。感兴趣的可以看`shareAI-lab/mini-claude-code`这个原始项目的实现。


### Agent的执行过程
- 1.发送请求,并接收响应
- 2.响应的StopReason如果不是工具调用就退出循环。
- 3.如果是工具调用就执行工具调用,继续循环


### Anthropic的工具调用的官方文档: 
- https://platform.claude.com/docs/en/agents-and-tools/tool-use/programmatic-tool-calling
- https://platform.claude.com/docs/en/api/go/messages/create
- https://platform.claude.com/docs/en/agents-and-tools/tool-use/tool-reference
