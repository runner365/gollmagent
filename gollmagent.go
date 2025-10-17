package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/gollmagent/ffmpegcmd"
	"github.com/gollmagent/llmproxy"
	log "github.com/gollmagent/logging"
	"github.com/gollmagent/progressmgr"
	"github.com/gollmagent/pub"
	"github.com/gollmagent/websocket"
)

var (
	logfile    = flag.String("logfile", "server.log", "Log file path")
	loglevel   = flag.String("loglevel", "info", "Log level :debug, info, warn, error, fatal.")
	wsPort     = flag.Int("wsport", 8080, "WebSocket server port")
	serverMode = flag.Bool("server", false, "Run in server mode")
	llmType    = flag.String("llmtype", "yuanbao", "LLM type: qwen, yuanbao")
)

var supportedLLMTypes map[string]pub.LLMTypeInfo

func init() {
	flag.Parse()
	log.SetOutputByName(*logfile)
	log.SetRotateByDay()
	log.SetLevelByString(*loglevel)

	initSupportedLLMTypes()
}

func initSupportedLLMTypes() {
	supportedLLMTypes = make(map[string]pub.LLMTypeInfo)
	supportedLLMTypes["qwen"] = pub.LLMTypeInfo{
		LLMType: "qwen",
		Url:     "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions",
		Model:   "qwen-plus",
	}
	supportedLLMTypes["yuanbao"] = pub.LLMTypeInfo{
		LLMType: "yuanbao",
		Url:     "https://api.hunyuan.cloud.tencent.com/v1/chat/completions",
		Model:   "hunyuan-turbo",
	}
}

func main() {
	log.Infof("Starting gollmagent...")
	for k, v := range supportedLLMTypes {
		log.Infof("LLM Type: %s, Url: %s, Model: %s", k, v.Url, v.Model)
	}
	llmInfo, ok := supportedLLMTypes[*llmType]
	if !ok {
		fmt.Printf("Unsupported llmtype: %s\n", *llmType)
		log.Fatalf("Unsupported llmtype: %s", *llmType)
	}

	// it's for qwen alibaba cloud, and it is required. you can change it if you use other llm service
	llmUrl := llmInfo.Url
	model := llmInfo.Model

	log.Infof("Using LLM Type: %s, Url: %s, Model: %s", *llmType, llmUrl, model)

	// LLM API Key is required for large language model access
	llmSecKey := os.Getenv("LLM_API_KEY")

	// APP_ID, SECRET_ID, SECRET_KEY is for tencent cloud, they are optional
	SecretId := os.Getenv("SECRET_ID")
	SecretKey := os.Getenv("SECRET_KEY")
	appId := os.Getenv("APP_ID")

	if len(llmSecKey) == 0 {
		fmt.Println("LLM_API_KEY environment variable is not set, exiting...")
		log.Fatal("LLM_API_KEY environment variable is not set")
	}
	if len(SecretId) == 0 {
		log.Info("LLM_SECRET_ID environment variable is not set")
	}
	if len(SecretKey) == 0 {
		log.Info("LLM_SECRET_KEY environment variable is not set")
	}

	go log.CrashLog("crash.log")

	//it's not required, if you don't use voice feature, you can set it to nil
	voiceAuth := &llmproxy.VoiceAuthInfo{
		AppId:     appId,
		SecretId:  SecretId,
		SecretKey: SecretKey,
	}
	desc := llmproxy.InitTools()
	fmt.Printf("Available tools:\n%s\n", desc)

	ffmpegVer := ffmpegcmd.GetFFmpegVersion()
	if len(ffmpegVer) == 0 {
		fmt.Println("ffmpeg not found or error")
		log.Fatalf("ffmpeg not found or error")
	}
	fmt.Printf("ffmpeg version: %s\n\n", ffmpegVer)
	log.Infof("ffmpeg version: %s\n\n", ffmpegVer)

	llmproxy.CreateFunctionToolsHandler()

	// create llm proxy object
	llmProxyObj := llmproxy.NewLLMProxy(llmUrl, model, llmSecKey, voiceAuth)

	// create progress manager, it will manage the progress of tools execution
	progressmgr := progressmgr.NewProgressMgr(llmProxyObj, llmproxy.FunctionTools)
	progressmgr.Run()

	// create websocket server, it will handle the chat messages from web clients(include text, voice)
	wsServ := websocket.NewWsServer(llmProxyObj)

	llmproxy.SetProgressCallback(progressmgr)

	http.HandleFunc("/chat", wsServ.HandleWebSocket)

	if *serverMode {
		http.ListenAndServe(fmt.Sprintf(":%d", *wsPort), nil)
		return
	} else {
		llmproxy.CommandRun2(llmProxyObj, progressmgr)
	}
}
