package llmproxy

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	log "github.com/gollmagent/logging"
	"github.com/gollmagent/pub"
)

func CommandRun2(ybObj *LLMProxy, progressCb pub.ProgressCallback) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("用户: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("\n检测到EOF，退出程序")
				break
			}
			fmt.Println("\n读取错误:", err)
			continue
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		if strings.EqualFold(input, "exit") || strings.EqualFold(input, "quit") {
			break
		}

		fmt.Println()
		log.Infof("User input: %s", input)

		resp, err := ybObj.ChatCompletions(input, FunctionTools)
		if err != nil {
			log.Errorf("AI回复时出错: %v", err)
			continue
		}
		if len(resp.Choices) == 0 {
			fmt.Println("\r\nAI: 没能力回复")
			continue
		}

		respText, err := handleRespMessage(ybObj, progressCb, input, resp)
		if err != nil {
			log.Errorf("处理AI回复时出错: %v", err)
			fmt.Println("\r\nAI: 出错了, ", err)
			continue
		}
		fmt.Printf("\r\nAI: %s\n\n", respText)
	}
}

func handleRespMessage(ybObj *LLMProxy, progressCb pub.ProgressCallback, input string, resp *pub.ChatCompletionsResponse) (string, error) {
	if resp == nil {
		return "", fmt.Errorf("response is nil")
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	for _, choice := range resp.Choices {
		if input == choice.Message.Content {
			continue
		}
		if choice.Message.Role == "assistant" {
			if len(choice.Message.ToolCalls) > 0 {
				callId := choice.Message.ToolCallID
				toolResp, err := handleToolsCall(progressCb, choice.Message.ToolCalls)
				if err != nil {
					log.Errorf("处理工具调用时出错: %v", err)
					return "", err
				}
				if len(callId) == 0 {
					callId = choice.Message.ToolCalls[0].ID
				}
				toolsResp, err := ybObj.ToolResultCompletions(toolResp, callId)
				if err != nil {
					log.Errorf("处理工具调用时出错: %v", err)
					return "", err
				}
				for _, tChoice := range toolsResp.Choices {
					if tChoice.Message.Role == "assistant" && len(tChoice.Message.Content) > 0 {
						return tChoice.Message.Content, nil
					}
				}
				return "", fmt.Errorf("no valid response from tool call")
			}
			return choice.Message.Content, nil
		}
	}
	return "", fmt.Errorf("no valid response found")
}
