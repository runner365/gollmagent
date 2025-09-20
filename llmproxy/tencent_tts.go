package llmproxy

import (
	"time"

	log "github.com/gollmagent/logging"
	"github.com/gollmagent/pub/ttscallback"
	"github.com/tencentcloud/tencentcloud-speech-sdk-go/common"
	"github.com/tencentcloud/tencentcloud-speech-sdk-go/tts"
)

var kVoiceType int64 = 1008

type TencentTTS struct {
	ID        string
	appId     int64
	secretId  string
	secretKey string
	cb        ttscallback.TtsCallbackI
	ttsObj    *tts.SpeechSynthesizer
}

func NewTencentTTS(id string, appId int64, secretId, secretKey string, cb ttscallback.TtsCallbackI) *TencentTTS {
	log.Infof("Creating new TencentTTS instance with ID: %s", id)
	return &TencentTTS{
		ID:        id,
		appId:     appId,
		secretId:  secretId,
		secretKey: secretKey,
		cb:        cb,
	}
}

func (txTts *TencentTTS) Start() error {
	if txTts.ttsObj != nil {
		return nil
	}
	credential := common.NewCredential(txTts.secretId, txTts.secretKey)
	txTts.ttsObj = tts.NewSpeechSynthesizer(txTts.appId, credential, txTts)
	txTts.ttsObj.VoiceType = kVoiceType
	log.Infof("Starting TencentTTS for ID: %s", txTts.ID)
	// Initialize TTS service here
	return nil
}

func (txTts *TencentTTS) Stop() {
	if txTts.ttsObj == nil {
		return
	}
	log.Infof("Stopping TencentTTS for ID: %s", txTts.ID)
	txTts.ttsObj = nil
}

func (txTts *TencentTTS) Write(text string) error {
	if txTts.ttsObj == nil {
		log.Error("speech synthesizer is not started")
		return nil
	}
	log.Infof("Writing text to TencentTTS: %s", text)
	err := txTts.ttsObj.Synthesis(text)
	if err != nil {
		log.Errorf("Failed to write text to TencentTTS: %v", err)
		return err
	}
	return nil
}

// OnMessage implementation of SpeechSynthesisListener
func (txTts *TencentTTS) OnMessage(response *tts.SpeechSynthesisResponse) {
	log.Infof("%s|%s|OnMessage, size: %d\n", time.Now().Format("2006-01-02 15:04:05"), txTts.ID, len(response.Data))
	if txTts.cb != nil {
		txTts.cb.OnText2Pcm(response.Data, txTts.ID, ttscallback.PcmDelta)
	}
}

// OnComplete implementation of SpeechSynthesisListener
func (txTts *TencentTTS) OnComplete(response *tts.SpeechSynthesisResponse) {
	log.Infof("%s|%s|OnComplete: %v\n", time.Now().Format("2006-01-02 15:04:05"), txTts.ID, response)
	if txTts.cb != nil {
		txTts.cb.OnText2Pcm(response.Data, txTts.ID, ttscallback.PcmDone)
	}
}

// OnCancel implementation of SpeechSynthesisListener
func (txTts *TencentTTS) OnCancel(response *tts.SpeechSynthesisResponse) {
	log.Infof("%s|%s|OnCancel: %v\n", time.Now().Format("2006-01-02 15:04:05"), txTts.ID, response)
}

// OnFail implementation of SpeechSynthesisListener
func (txTts *TencentTTS) OnFail(response *tts.SpeechSynthesisResponse, err error) {
	log.Infof("%s|%s|OnFail: %v, %v\n", time.Now().Format("2006-01-02 15:04:05"), txTts.ID, response, err)
	if txTts.cb != nil {
		txTts.cb.OnTtsError(err, txTts.ID)
	}
}
