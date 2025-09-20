package llmproxy

import (
	"fmt"
	"time"

	log "github.com/gollmagent/logging"
	"github.com/gollmagent/pub/asrcallback"
	"github.com/tencentcloud/tencentcloud-speech-sdk-go/asr"
	"github.com/tencentcloud/tencentcloud-speech-sdk-go/common"
)

type TencentASR struct {
	ID        string
	VoiceID   string
	appId     string
	secretId  string
	secretKey string
	asrObj    *asr.SpeechRecognizer
	cb        asrcallback.ASRCallbackI
}

func NewTencentASR(id string, appId, secretId, secretKey string, cb asrcallback.ASRCallbackI) *TencentASR {
	log.Infof("Creating new TencentASR instance with ID: %s", id)
	return &TencentASR{
		ID:        id,
		appId:     appId,
		secretId:  secretId,
		secretKey: secretKey,
		cb:        cb,
	}
}

func (txAsr *TencentASR) Start() error {
	if txAsr.asrObj != nil {
		return nil
	}
	credential := common.NewCredential(txAsr.secretId, txAsr.secretKey)
	txAsr.asrObj = asr.NewSpeechRecognizer(txAsr.appId, credential, kEngineModelType, txAsr)
	err := txAsr.asrObj.Start()
	if err != nil {
		txAsr.asrObj = nil
		return err
	}
	log.Infof("TencentASR started for ID: %s", txAsr.ID)
	return nil
}

func (txAsr *TencentASR) Stop() {
	if txAsr.asrObj == nil {
		return
	}
	log.Infof("Stopping TencentASR for ID: %s", txAsr.ID)
	txAsr.asrObj.Stop()
	txAsr.asrObj = nil
}

func (txAsr *TencentASR) Write(voiceData []byte) error {
	if txAsr.asrObj == nil {
		log.Error("speech recognizer is not started")
		return fmt.Errorf("speech recognizer is not started")
	}
	return txAsr.asrObj.Write(voiceData)
}

// OnRecognitionStart implementation of SpeechRecognitionListener
func (txAsr *TencentASR) OnRecognitionStart(response *asr.SpeechRecognitionResponse) {
	txAsr.VoiceID = response.VoiceID
	log.Infof("%s|%s|OnRecognitionStart\n", time.Now().Format("2006-01-02 15:04:05"), response.VoiceID)
}

// OnSentenceBegin implementation of SpeechRecognitionListener
func (txAsr *TencentASR) OnSentenceBegin(response *asr.SpeechRecognitionResponse) {
	log.Infof("%s|%s|OnSentenceBegin: %v\n", time.Now().Format("2006-01-02 15:04:05"), response.VoiceID, response)
}

// OnRecognitionResultChange implementation of SpeechRecognitionListener
func (txAsr *TencentASR) OnRecognitionResultChange(response *asr.SpeechRecognitionResponse) {
	log.Infof("%s|%s|OnRecognitionResultChange: %v\n", time.Now().Format("2006-01-02 15:04:05"), response.VoiceID, response)
}

// OnSentenceEnd implementation of SpeechRecognitionListener
func (txAsr *TencentASR) OnSentenceEnd(response *asr.SpeechRecognitionResponse) {
	log.Infof("%s|%s|OnSentenceEnd: %+v\n", time.Now().Format("2006-01-02 15:04:05"), response.VoiceID, response)
	if txAsr.cb != nil {
		txAsr.cb.OnAsr2Text(response.Result.VoiceTextStr, response.VoiceID)
	}
}

// OnRecognitionComplete implementation of SpeechRecognitionListener
func (txAsr *TencentASR) OnRecognitionComplete(response *asr.SpeechRecognitionResponse) {
	log.Infof("%s|%s|OnRecognitionComplete\n", time.Now().Format("2006-01-02 15:04:05"), response.VoiceID)
	if txAsr.cb != nil {
		txAsr.cb.OnAsrEnd(response.VoiceID)
	}
}

// OnFail implementation of SpeechRecognitionListener
func (txAsr *TencentASR) OnFail(response *asr.SpeechRecognitionResponse, err error) {
	log.Errorf("%s|%s|OnFail: %v\n", time.Now().Format("2006-01-02 15:04:05"), response.VoiceID, err)
	if txAsr.cb != nil {
		txAsr.cb.OnAsrError(err, response.VoiceID)
	}
}
