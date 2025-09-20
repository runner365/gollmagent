package progressmgr

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/gollmagent/logging"
	"github.com/gollmagent/pub"
	"github.com/gollmagent/llmproxy"
)

type ProgressMgr struct {
	llm         pub.LlmProxyInterface
	progressMap map[string]*pub.ProgressInfo
	tools       []*pub.ToolDefinition
	mutex       sync.Mutex
	closeChan   chan int
}

func NewProgressMgr(llm pub.LlmProxyInterface, tools []*pub.ToolDefinition) *ProgressMgr {
	return &ProgressMgr{
		llm:         llm,
		progressMap: make(map[string]*pub.ProgressInfo),
		tools:       tools,
		closeChan:   make(chan int, 100),
	}
}

func (mgr *ProgressMgr) Run() error {
	go mgr.onWork()
	return nil
}

func (mgr *ProgressMgr) OnProgress(info *pub.ProgressInfo, id string) {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()
	log.Infof("Progress update for ID=%s, info:%v", id, info)
	mgr.progressMap[id] = info
}

func (mgr *ProgressMgr) onWork() {
	for {
		select {
		case <-mgr.closeChan:
			log.Info("ProgressMgr received close signal, exiting onWork loop")
			return
		case <-time.After(2 * time.Second):
			mgr.onCheckProgress()
		}
	}
}

func (mgr *ProgressMgr) onCheckProgress() {
	nowMs := time.Now().UnixMilli()

	var removeIds []string
	for id, info := range mgr.progressMap {
		diff := nowMs - int64(info.Ms)
		if diff < 5000 {
			continue
		}
		info.Ms = uint64(nowMs)
		msg := fmt.Sprintf("请查询任务进度, task_id:%s", id)
		log.Infof("Checking progress for ID=%s, info:%v, post:'%s'", id, info, msg)
		resp, err := mgr.llm.ChatCompletions(msg, mgr.tools)
		if err != nil {
			log.Errorf("Error checking progress for ID=%s: %v", id, err)
			continue
		}
		done, err := mgr.handleResponse(resp)
		if err != nil {
			log.Errorf("Error handling response for ID=%s: %v", id, err)
			continue
		}
		if done {
			log.Infof("Progress for ID=%s is done", id)
			removeIds = append(removeIds, id)
		}
	}
	for _, id := range removeIds {
		log.Infof("Removing completed progress ID=%s", id)
		mgr.mutex.Lock()
		delete(mgr.progressMap, id)
		mgr.mutex.Unlock()
	}
}

func (mgr *ProgressMgr) handleResponse(resp *pub.ChatCompletionsResponse) (bool, error) {
	if resp == nil || len(resp.Choices) == 0 {
		return false, fmt.Errorf("empty response or no choices")
	}
	done := false
	for _, choice := range resp.Choices {
		if choice.Message.Role == "assistant" {
			if choice.Message.ToolCalls != nil && len(choice.Message.ToolCalls) > 0 {
				log.Infof("Tool call response: %+v", choice.Message.ToolCalls)
				for _, toolCall := range choice.Message.ToolCalls {
					if toolCall.Function.Name == "check_progress" {
						checkProgressFunction := llmproxy.GetToolFunctionByName("check_progress")
						if checkProgressFunction == nil {
							log.Errorf("check_progress tool not found")
							continue
						}
						argsStr, ok := toolCall.Function.Arguments.(string)
						if ok {
							log.Infof("check_progress tool call arguments: %s", argsStr)
							args := make(map[string]interface{})
							err := json.Unmarshal([]byte(argsStr), &args)
							if err != nil {
								log.Errorf("Error unmarshaling tool call arguments: %v", err)
								continue
							}
							resp := checkProgressFunction(args)
							log.Infof("check_progress tool call result: %v", resp)
							respStr, ok := resp.(string)
							if !ok {
								log.Errorf("check_progress tool call result is not string")
								continue
							}
							index := strings.Index(respStr, "完成")
							if index >= 0 {
								done = true
							}
							index = strings.Index(respStr, "done")
							if index >= 0 {
								done = true
							}
							toolResp, err := mgr.llm.ToolResultCompletions(respStr, toolCall.ID)
							if err != nil {
								log.Errorf("Error sending tool result completions: %v", err)
								continue
							}
							for _, tChoice := range toolResp.Choices {
								if tChoice.Message.Role == "assistant" && len(tChoice.Message.Content) > 0 {
									fmt.Println("\r\nAI: ", tChoice.Message.Content)
								}
							}
						}
					}
				}
			}
		}
	}
	return done, nil
}

func (mgr *ProgressMgr) CheckProgress(id string) *pub.ProgressInfo {
	info, exists := mgr.progressMap[id]
	if !exists {
		return nil
	}
	return info
}
