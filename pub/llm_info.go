package pub

type ChatCompletionsInfo struct {
	Model      string                   `json:"model"`
	Messages   []ChatCompletionsMessage `json:"messages"`
	Tools      []ToolDefinition         `json:"tools,omitempty"`       // 可选字段，支持工具调用
	ToolChoice *ToolChoice              `json:"tool_choice,omitempty"` // 可选字段，控制工具调用行为
}

type Function func(args map[string]interface{}) interface{}

// ToolCall 表示一个工具调用（tool call），通常是函数调用
type ToolCall struct {
	ID       string       `json:"id"`   // 唯一标识，比如 "call_abc123"
	Type     string       `json:"type"` // 固定为 "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall 表示调用了哪个函数，以及传入了哪些参数
type FunctionCall struct {
	Name      string      `json:"name"`      // 函数名，如 "get_current_weather"
	Arguments interface{} `json:"arguments"` // 函数的参数，通常是 JSON 对象（键值对），也可能是字符串
}

type ChatCompletionsMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCallID string     `json:"tool_call_id,omitempty"` // 当模型调用工具时，返回 tool_call_id
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // 当模型调用工具时，返回 tool_calls
}

type ChatCompletionsResponse struct {
	ID      string                  `json:"id"`
	Object  string                  `json:"object"`
	Created int64                   `json:"created"`
	Model   string                  `json:"model"`
	Choices []ChatCompletionsChoice `json:"choices"`
	Usage   TokensUsage             `json:"usage"`
	Note    string                  `json:"note,omitempty"`
}

type ChatCompletionsChoice struct {
	Index        int                    `json:"index"`
	Message      ChatCompletionsMessage `json:"message"`
	FinishReason string                 `json:"finish_reason"`
}

type TokensUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

/************* tool functions *************/
// ToolDefinition 对应 OpenAI API 中的 tools 数组中的每一项
type ToolDefinition struct {
	Type     string             `json:"type"`     // 目前只支持 "function"
	Function FunctionDefinition `json:"function"` // 工具的具体定义
}

type FunctionDefinition struct {
	Name        string                 `json:"name"`                  // 函数名
	Description string                 `json:"description,omitempty"` // 函数描述
	Parameters  map[string]interface{} `json:"parameters"`            // 参数 JSON Schema（可以是对象结构，通常用 map 或具体结构体）
}

// ToolChoice 控制是否以及如何调用工具
type ToolChoice struct {
	Type string `json:"type"` // 可选值："auto", "none", 或 "tool"（具体工具）
	Tool *struct {
		Function FunctionReference `json:"function"` // 当 type == "tool" 时指定具体函数
	} `json:"tool,omitempty"` // 仅当 type == "tool" 时使用
}

type FunctionReference struct {
	Name string `json:"name"` // 指定要调用的函数名
}

/************* end of tool functions *************/

type LlmProxyInterface interface {
	ChatCompletions(prompt string, tools []*ToolDefinition) (*ChatCompletionsResponse, error)
	ToolResultCompletions(text string, callId string) (*ChatCompletionsResponse, error)
}
