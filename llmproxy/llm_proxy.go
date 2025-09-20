package llmproxy

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gollmagent/httpclient"
	log "github.com/gollmagent/logging"
	"github.com/gollmagent/pub"
	"github.com/gollmagent/pub/ttscallback"
	"github.com/gollmagent/utils"
)

const kMessageMax = 20
const kEngineModelType = "16k_zh"
const kVoiceChannMax = 1000
const kSendBufferSize = 1000

type VoiceAuthInfo struct {
	AppId     string
	SecretId  string
	SecretKey string
}

type LLMProxy struct {
	llmUrl	    string
	model	   	string
	llmSecKey   string
	voiceAuth   *VoiceAuthInfo
	messages    []pub.ChatCompletionsMessage
	msgMutex    sync.Mutex // Ensure thread-safe access to messages
	voiceChann  chan *pub.ChatVoiceInfo
	sendChann   chan []byte
	userId      string
	currentWs   pub.WsStreamI
	asrHandlers map[string]*TencentASR
	asrMutex    sync.Mutex // Ensure thread-safe access to ASR handlers
	ttsHandlers map[string]*TencentTTS
	ttsMutex    sync.Mutex // Ensure thread-safe access to TTS handlers
}

func NewLLMProxy(llmUrl string, model string, llmSecKey string, voiceAuth *VoiceAuthInfo) *LLMProxy {
	log.Infof("Creating new LLMProxy instance with llmUrl: %s, model: %s", llmUrl, model)
	ret := &LLMProxy{
		llmUrl:     llmUrl,
		model:       model,
		llmSecKey:  llmSecKey,
		voiceAuth:   voiceAuth,
		voiceChann:  make(chan *pub.ChatVoiceInfo, kVoiceChannMax),
		sendChann:   make(chan []byte, kSendBufferSize),
		asrHandlers: make(map[string]*TencentASR),
		ttsHandlers: make(map[string]*TencentTTS),
	}
	go ret.onReceiveVoice()
	go ret.onSendData()
	return ret
}

func (proxy *LLMProxy) addTtsHandler(id string, ttsHandler *TencentTTS) {
	proxy.ttsMutex.Lock()
	defer proxy.ttsMutex.Unlock()
	log.Infof("Adding TTS handler for id: %s", id)
	proxy.ttsHandlers[id] = ttsHandler
}

func (proxy *LLMProxy) getTtsHandler(id string) *TencentTTS {
	proxy.ttsMutex.Lock()
	defer proxy.ttsMutex.Unlock()
	if handler, exists := proxy.ttsHandlers[id]; exists {
		return handler
	}
	log.Errorf("TTS handler not found for id: %s", id)
	return nil
}

func (proxy *LLMProxy) removeTtsHandler(id string) {
	proxy.ttsMutex.Lock()
	defer proxy.ttsMutex.Unlock()
	if _, exists := proxy.ttsHandlers[id]; exists {
		log.Infof("Removing TTS handler for id: %s", id)
		delete(proxy.ttsHandlers, id)
	} else {
		log.Errorf("TTS handler not found for id: %s", id)
	}
}

func (proxy *LLMProxy) addAsrHandler(id string, asrHandler *TencentASR) {
	proxy.asrMutex.Lock()
	defer proxy.asrMutex.Unlock()
	log.Infof("Adding ASR handler for id: %s", id)
	proxy.asrHandlers[id] = asrHandler
}

func (proxy *LLMProxy) getAsrHandler(id string) *TencentASR {
	proxy.asrMutex.Lock()
	defer proxy.asrMutex.Unlock()
	if handler, exists := proxy.asrHandlers[id]; exists {
		return handler
	}
	log.Errorf("ASR handler not found for id: %s", id)
	return nil
}

func (proxy *LLMProxy) getAsrHandlerByAudioId(audioId string) *TencentASR {
	proxy.asrMutex.Lock()
	defer proxy.asrMutex.Unlock()
	for _, handler := range proxy.asrHandlers {
		if handler.VoiceID == audioId {
			return handler
		}
	}
	log.Errorf("ASR handler not found for audioId: %s", audioId)
	return nil
}

func (proxy *LLMProxy) removeAsrHandler(id string) {
	proxy.asrMutex.Lock()
	defer proxy.asrMutex.Unlock()
	if _, exists := proxy.asrHandlers[id]; exists {
		log.Infof("Removing ASR handler for id: %s", id)
		delete(proxy.asrHandlers, id)
	} else {
		log.Errorf("ASR handler not found for id: %s", id)
	}
}

func (proxy *LLMProxy) clearMessages() {
	proxy.msgMutex.Lock()
	defer proxy.msgMutex.Unlock()
	proxy.messages = []pub.ChatCompletionsMessage{}
}

func (proxy *LLMProxy) addMessage(msg *pub.ChatCompletionsMessage) {
	proxy.msgMutex.Lock()
	defer proxy.msgMutex.Unlock()

	if len(proxy.messages)+1 >= kMessageMax {
		proxy.messages = proxy.messages[1:]
	}
	proxy.messages = append(proxy.messages, *msg)
}

func (proxy *LLMProxy) getMessages() []pub.ChatCompletionsMessage {
	proxy.msgMutex.Lock()
	defer proxy.msgMutex.Unlock()

	messagesCopy := make([]pub.ChatCompletionsMessage, len(proxy.messages))
	copy(messagesCopy, proxy.messages)
	return messagesCopy
}

func (proxy *LLMProxy) ToolResultCompletions(text string, callId string) (*pub.ChatCompletionsResponse, error) {
	secretKey := proxy.llmSecKey

	isHttps, hostname, port, subpath, err := utils.ParseURL(proxy.llmUrl)
	if err != nil {
		log.Errorf("Failed to parse LLM URL: %v", err)
		return nil, err
	}

	if !isHttps {
		log.Errorf("Only https is supported for llmUrl, url: %s", proxy.llmUrl)
		return nil, fmt.Errorf("only https is supported for llmUrl")
	}
	newMsg := &pub.ChatCompletionsMessage{
		Role:       "tool",
		ToolCallID: callId,
		Content:    text,
	}
	proxy.addMessage(newMsg)

	msgs := proxy.getMessages()

	info := &pub.ChatCompletionsInfo{
		Model:    proxy.model,
		Messages: msgs,
	}
	for _, tool := range FunctionTools {
		info.Tools = append(info.Tools, *tool)
	}
	jsonData, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}
	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	header.Set("Authorization", "Bearer "+secretKey)

	log.Infof("Sending request to %s:%d%s with data: %s", hostname, port, subpath, string(jsonData))
	log.Infof("header: %v", header)

	respData, err := httpclient.HTTPSPost(hostname, port, subpath, jsonData, header)
	if err != nil {
		return nil, err
	}
	proxy.clearMessages()
	log.Infof("tool result Response data: %s", string(respData))

	resp := &pub.ChatCompletionsResponse{}
	if err := json.Unmarshal(respData, resp); err != nil {
		log.Errorf("Failed to unmarshal response: %v", err)
		return nil, err
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices found in response")
	}
	return resp, nil
}

func (proxy *LLMProxy) ChatCompletions(prompt string, FunctionTools []*pub.ToolDefinition) (*pub.ChatCompletionsResponse, error) {
	secretKey := proxy.llmSecKey

	isHttps, hostname, port, subpath, err := utils.ParseURL(proxy.llmUrl)
	if err != nil {
		log.Errorf("Failed to parse LLM URL: %v", err)
		return nil, err
	}
	if !isHttps {
		log.Errorf("Only https is supported for llmUrl, url: %s", proxy.llmUrl)
		return nil, fmt.Errorf("only https is supported for llmUrl")
	}

	prompt = fmt.Sprintf("%s, 回答请简短，并且消息不使用markdown格式", prompt)
	newMsg := &pub.ChatCompletionsMessage{
		Role:    "user",
		Content: prompt,
	}
	proxy.addMessage(newMsg)

	msgs := proxy.getMessages()

	info := &pub.ChatCompletionsInfo{
		Model:    proxy.model,
		Messages: msgs,
	}
	for _, tool := range FunctionTools {
		info.Tools = append(info.Tools, *tool)
	}
	jsonData, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}
	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	header.Set("Authorization", "Bearer "+secretKey)

	log.Infof("Sending request to %s:%d%s with data: %s", hostname, port, subpath, string(jsonData))
	log.Infof("header: %v", header)

	respData, err := httpclient.HTTPSPost(hostname, port, subpath, jsonData, header)
	if err != nil {
		log.Errorf("Failed to send request: %v, resp: %s", err, string(respData))
		return nil, err
	}
	log.Infof("Response data: %s", string(respData))

	resp := &pub.ChatCompletionsResponse{}
	if err := json.Unmarshal(respData, resp); err != nil {
		log.Errorf("Failed to unmarshal response: %v", err)
		return nil, err
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices found in response")
	}
	for _, choice := range resp.Choices {
		if choice.Message.Role == "assistant" {
			proxy.addMessage(&choice.Message)
		}
	}
	return resp, nil
}

func (proxy *LLMProxy) handleClientMessage(info *pub.ChatMessageInfo, ws pub.WsStreamI) error {
	resp, err := proxy.ChatCompletions(info.Content, nil)
	if err != nil {
		log.Errorf("chatCompletions failed: %v", err)
		return nil
	}
	log.Infof("chatCompletions response: %v", resp)

	if len(resp.Choices) == 0 {
		log.Errorf("No choices found in response")
		return nil
	}
	for _, choice := range resp.Choices {
		if choice.Message.Role == "assistant" {
			content := utils.MarkdownToText(choice.Message.Content)
			msg := &pub.ChatMessageInfo{
				MsgType: "chat.completions",
				UserId:  "ai",
				Role:    "assistant",
				ItemId:  info.ItemId,
				Content: content,
				Ts:      time.Now().UnixMilli(),
			}
			jsonData, err := json.Marshal(msg)
			if err != nil {
				log.Errorf("Failed to marshal message: %v", err)
				return err
			}
			log.Infof("Sending message to WebSocket: %s", string(jsonData))

			go func() {
				ttsObj := proxy.getTtsHandler(info.ItemId)
				if ttsObj == nil {
					log.Errorf("TTS handler not found for itemId: %s", info.ItemId)
					appIdInt64, err := strconv.ParseInt(proxy.voiceAuth.AppId, 10, 64)
					if err != nil {
						log.Errorf("Failed to convert AppId to int64: %v", err)
						return
					}
					ttsObj = NewTencentTTS(info.ItemId, appIdInt64, proxy.voiceAuth.SecretId, proxy.voiceAuth.SecretKey, proxy)
					if err := ttsObj.Start(); err != nil {
						log.Errorf("Failed to start TTS handler: %v", err)
						return
					}
					proxy.addTtsHandler(info.ItemId, ttsObj)
				}
				if err := ttsObj.Write(choice.Message.Content); err != nil {
					log.Errorf("Failed to write text to TTS handler: %v", err)
					return
				}
			}()
			proxy.sendChann <- jsonData
		}
	}
	return nil
}

func (proxy *LLMProxy) handleClientVoiceMessage(info *pub.ChatVoiceInfo) error {
	var decodedData []byte
	var err error

	log.Debugf("Received voice message: %+v", info)

	if len(info.Base64Data) > 0 {
		decodedData, err = base64.StdEncoding.DecodeString(info.Base64Data)
		if err != nil {
			log.Errorf("Failed to decode base64 data: %v", err)
			return err
		}
	}

	asrHandler := proxy.getAsrHandler(info.ItemId)
	if asrHandler == nil {
		log.Errorf("ASR handler not found for itemId: %s", info.ItemId)
		asrHandler = NewTencentASR(info.ItemId, proxy.voiceAuth.AppId, proxy.voiceAuth.SecretId, proxy.voiceAuth.SecretKey, proxy)
		if err := asrHandler.Start(); err != nil {
			log.Errorf("Failed to start ASR handler: %v", err)
			return err
		}
		proxy.addAsrHandler(info.ItemId, asrHandler)
	}

	if len(decodedData) > 0 {
		asrHandler.Write(decodedData)
	}

	if info.AudioType == "done" {
		asrHandler.Stop()
	}
	return nil
}

func (proxy *LLMProxy) insertVoiceMessage2Chann(voiceInfo *pub.ChatVoiceInfo) error {
	defer func() {
		if rc := recover(); rc != nil {
			log.Errorf("Recovered from panic: %v", rc)
		}
	}()

	if proxy.voiceChann == nil {
		log.Errorf("Voice channel is nil")
		return fmt.Errorf("voice channel is nil")
	}
	if len(proxy.voiceChann) >= kVoiceChannMax {
		log.Errorf("Voice channel is full, dropping message")
		return fmt.Errorf("voice channel is full")
	}

	proxy.voiceChann <- voiceInfo
	return nil
}

func (proxy *LLMProxy) onReceiveVoice() {
	for {
		voiceInfo, ret := <-proxy.voiceChann
		if !ret {
			log.Errorf("Voice channel closed")
			return
		}

		if err := proxy.handleClientVoiceMessage(voiceInfo); err != nil {
			log.Errorf("Failed to handle client voice message: %v", err)
		}
	}
}

func (proxy *LLMProxy) sendMessageToWebSocket(msg []byte) {
	defer func() {
		if rc := recover(); rc != nil {
			log.Errorf("Recovered from panic in sendMessageToWebSocket: %v", rc)
		}
	}()
	if proxy.currentWs == nil {
		log.Error("Current WebSocket is nil, cannot send message")
		return
	}
	proxy.currentWs.Send(msg)
}

func (proxy *LLMProxy) onSendData() {
	for {
		data, ret := <-proxy.sendChann
		if !ret {
			log.Errorf("Send channel closed")
			return
		}

		proxy.sendMessageToWebSocket(data)
	}
}

func (proxy *LLMProxy) OnMessage(id string, data []byte, ws pub.WsStreamI) error {
	defer func() {
		if rc := recover(); rc != nil {
			log.Errorf("Recovered from panic: %v", rc)
		}
	}()
	info := &pub.ChatMessageBaseInfo{}
	if proxy.currentWs != ws {
		proxy.currentWs = ws
	}
	err := json.Unmarshal(data, info)
	if err != nil {
		log.Errorf("Failed to unmarshal WebSocket message: %v", err)
		return err
	}
	log.Debugf("Received WebSocket message: %+v", info)

	switch info.MsgType {
	case "chat.completions":
		chatInfo := &pub.ChatMessageInfo{}
		err = json.Unmarshal(data, chatInfo)
		if err != nil {
			log.Errorf("Failed to unmarshal chat message: %v", err)
			return err
		}
		proxy.userId = chatInfo.UserId
		go proxy.handleClientMessage(chatInfo, ws)
	case "chat.voice":
		voiceInfo := &pub.ChatVoiceInfo{}
		err = json.Unmarshal(data, voiceInfo)
		if err != nil {
			log.Errorf("Failed to unmarshal voice message: %v", err)
			return err
		}
		proxy.userId = voiceInfo.UserId
		proxy.insertVoiceMessage2Chann(voiceInfo)
	default:
		log.Warningf("Unknown message type: %s, data:%s", info.MsgType, string(data))
		return fmt.Errorf("unknown message type: %s", info.MsgType)
	}

	return nil
}

func (proxy *LLMProxy) OnClose() error {
	log.Infof("WebSocket connection closed")
	return nil
}

func (proxy *LLMProxy) OnAsr2Text(text string, voiceId string) {
	asrHandler := proxy.getAsrHandlerByAudioId(voiceId)
	if asrHandler == nil {
		log.Errorf("ASR handler not found for voiceId: %s", voiceId)
		return
	}
	msgInfo := &pub.ChatMessageInfo{
		MsgType: "chat.completions",
		UserId:  proxy.userId,
		Role:    "user",
		ItemId:  asrHandler.ID,
		Content: text,
		Ts:      time.Now().UnixMilli(),
	}
	jsonData, err := json.Marshal(msgInfo)
	if err != nil {
		log.Errorf("Failed to marshal ASR text message: %v", err)
		return
	}
	log.Infof("Sending ASR text message: %s", string(jsonData))
	proxy.sendChann <- jsonData

	go proxy.voiceText2llm(text, asrHandler.ID, true)
}

func (proxy *LLMProxy) OnAsrEnd(id string) {
	log.Debugf("ASR ended, id: %s", id)
}

func (proxy *LLMProxy) OnAsrError(err error, id string) {
	log.Errorf("ASR error: %v, id: %s", err, id)
}

func (proxy *LLMProxy) voiceText2llm(text string, itemId string, ttsEnable bool) {
	resp, err := proxy.ChatCompletions(text, nil)
	if err != nil {
		log.Errorf("voice text call chatCompletions failed: %v", err)
		return
	}
	log.Infof("voice text chatCompletions response: %v", resp)
	if len(resp.Choices) == 0 {
		log.Errorf("No choices found in response")
		return
	}
	for _, choice := range resp.Choices {
		if choice.Message.Role == "assistant" {
			content := utils.MarkdownToText(choice.Message.Content)
			msg := &pub.ChatMessageInfo{
				MsgType: "chat.completions",
				UserId:  "ai",
				Role:    "assistant",
				ItemId:  itemId,
				Content: content,
				Ts:      time.Now().UnixMilli(),
			}
			jsonData, err := json.Marshal(msg)
			if err != nil {
				log.Errorf("Failed to marshal message: %v", err)
				return
			}
			log.Infof("voice response Sending message to WebSocket: %s", string(jsonData))

			if ttsEnable {
				ttsObj := proxy.getTtsHandler(itemId)
				if ttsObj == nil {
					log.Errorf("TTS handler not found for itemId: %s", itemId)
					appIdInt64, err := strconv.ParseInt(proxy.voiceAuth.AppId, 10, 64)
					if err != nil {
						log.Errorf("Failed to convert AppId to int64: %v", err)
						return
					}
					ttsObj = NewTencentTTS(itemId, appIdInt64, proxy.voiceAuth.SecretId, proxy.voiceAuth.SecretKey, proxy)
					if err := ttsObj.Start(); err != nil {
						log.Errorf("Failed to start TTS handler: %v", err)
						return
					}
					proxy.addTtsHandler(itemId, ttsObj)
				}
				ttsObj.Write(choice.Message.Content)
			}
			proxy.sendChann <- jsonData
		}
	}
}

func (proxy *LLMProxy) OnText2Pcm(pcmData []byte, id string, pcmType ttscallback.TtsPcmType) {
	defer func() {
		if rc := recover(); rc != nil {
			log.Errorf("Recovered from panic in OnText2Pcm: %v", rc)
		}
	}()
	utils.AppendToFile(fmt.Sprintf("tts_%s.pcm", id), pcmData)
	done := false
	if pcmType == ttscallback.PcmDone {
		done = true
	}
	voiceMsg := &pub.ChatMessageInfo{
		MsgType: "chat.voice",
		UserId:  "ai",
		ItemId:  id,
		Role:    "assistant",
		Content: base64.StdEncoding.EncodeToString(pcmData),
		Ts:      time.Now().UnixMilli(),
		Done:    done,
	}

	jsonData, err := json.Marshal(voiceMsg)
	if err != nil {
		log.Errorf("Failed to marshal voice message: %v", err)
		return
	}

	proxy.sendChann <- jsonData

	if pcmType == ttscallback.PcmDone {
		go func() {
			<-time.After(100 * time.Millisecond) // Ensure TTS handler is ready
			ttsHandler := proxy.getTtsHandler(id)
			if ttsHandler == nil {
				log.Errorf("TTS handler not found for id: %s", id)
				return
			}
			ttsHandler.Stop()
			proxy.removeTtsHandler(id)
		}()
	}
}

func (proxy *LLMProxy) OnTtsError(err error, id string) {
	log.Errorf("TTS error: %v, id: %s", err, id)
	ttsHandler := proxy.getTtsHandler(id)
	if ttsHandler != nil {
		ttsHandler.Stop()
		proxy.removeTtsHandler(id)
	}
}
