package asr

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"runtime/debug"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/tencentcloud/tencentcloud-speech-sdk-go/common"
)

// SpeechRecognitionListener User must impletement it. Get recognition result
type SpeechRecognitionListener interface {
	OnRecognitionStart(*SpeechRecognitionResponse)
	OnSentenceBegin(*SpeechRecognitionResponse)
	OnRecognitionResultChange(*SpeechRecognitionResponse)
	OnSentenceEnd(*SpeechRecognitionResponse)
	OnRecognitionComplete(*SpeechRecognitionResponse)
	OnFail(*SpeechRecognitionResponse, error)
}

// SpeechRecognitionResponse is the reponse of asr service
type SpeechRecognitionResponse struct {
	Code      int                             `json:"code"`
	Message   string                          `json:"message"`
	VoiceID   string                          `json:"voice_id,omitempty"`
	MessageID string                          `json:"message_id,omitempty"`
	Final     uint32                          `json:"final,omitempty"`
	Result    SpeechRecognitionResponseResult `json:"result,omitempty"`
}

// SpeechRecognitionResponseResult SpeechRecognitionResponseResult
type SpeechRecognitionResponseResult struct {
	SliceType    uint32                                `json:"slice_type"`
	Index        int                                   `json:"index"`
	StartTime    uint32                                `json:"start_time"`
	EndTime      uint32                                `json:"end_time"`
	VoiceTextStr string                                `json:"voice_text_str"`
	WordSize     uint32                                `json:"word_size"`
	WordList     []SpeechRecognitionResponseResultWord `json:"word_list"`
}

// SpeechRecognitionResponseResultWord SpeechRecognitionResponseResultWord
type SpeechRecognitionResponseResultWord struct {
	Word       string `json:"word"`
	StartTime  uint32 `json:"start_time"`
	EndTime    uint32 `json:"end_time"`
	StableFlag uint32 `json:"stable_flag"`
}

// AudioFormat type
const (
	AudioFormatPCM   = 1
	AudioFormatSpeex = 4
	AudioFormatSilk  = 6
	AudioFormatMp3   = 8
	AudioFormatOpus  = 10
	AudioFormatWav   = 12
	AudioFormatM4A   = 14
	AudioFormatAAC   = 16
)

// SpeechRecognizer is the entry for ASR service
type SpeechRecognizer struct {
	//request params
	AppID             string
	EngineModelType   string
	VoiceFormat       int
	NeedVad           int
	HotwordId         string
	HotwordList       string
	CustomizationId   string
	FilterDirty       int
	FilterModal       int
	FilterPunc        int
	ConvertNumMode    int
	WordInfo          int
	VadSilenceTime    int
	ReinforceHotword  int
	NoiseThreshold    float64
	FilterEmptyResult int
	MaxSpeakTime      int
	ReplaceTextId     string

	Credential *common.Credential
	//listener
	listener SpeechRecognitionListener
	//uuid for voice
	VoiceID string

	//for proxy
	ProxyURL string

	//for websocet connection
	conn *websocket.Conn

	//send data channel
	dataChan chan []byte
	//for listener get response message
	eventChan chan speechRecognitionEvent

	//used in stop function, waiting for stop all goroutines
	sendEnd    chan int
	receiveEnd chan int
	eventEnd   chan int

	mutex   sync.Mutex
	started bool
}

const (
	defaultVoiceFormat       = 1
	defaultNeedVad           = 1
	defaultWordInfo          = 0
	defaultFilterDirty       = 0
	defaultFilterModal       = 0
	defaultFilterPunc        = 0
	defaultConvertNumMode    = 1
	defaultReinforceHotword  = 0
	defaultFilterEmptyResult = 1
	defaultMaxSpeakTime      = 0

	protocol = "wss"
	host     = "asr.cloud.tencent.com"
	path     = ""
)

const (
	eventTypeRecognitionStart        = 0
	eventTypeSentenceBegin           = 1
	eventTypeRecognitionResultChange = 2
	eventTypeSentenceEnd             = 3
	eventTypeRecognitionComplete     = 4
	eventTypeFail                    = 5
)

type eventType int

type speechRecognitionEvent struct {
	t   eventType
	r   *SpeechRecognitionResponse
	err error
}

// NewSpeechRecognizer creates instance of SpeechRecognizer
func NewSpeechRecognizer(appID string, credential *common.Credential, engineModelType string,
	listener SpeechRecognitionListener) *SpeechRecognizer {

	reco := &SpeechRecognizer{
		AppID:             appID,
		Credential:        credential,
		EngineModelType:   engineModelType,
		VoiceFormat:       defaultVoiceFormat,
		NeedVad:           defaultNeedVad,
		FilterDirty:       defaultFilterDirty,
		FilterModal:       defaultFilterModal,
		FilterPunc:        defaultFilterPunc,
		ConvertNumMode:    defaultConvertNumMode,
		WordInfo:          defaultWordInfo,
		ReinforceHotword:  defaultReinforceHotword,
		FilterEmptyResult: defaultFilterEmptyResult,
		MaxSpeakTime:      defaultMaxSpeakTime,

		dataChan:  make(chan []byte, 6400),
		eventChan: make(chan speechRecognitionEvent, 10),

		sendEnd:    make(chan int),
		receiveEnd: make(chan int),
		eventEnd:   make(chan int),

		listener: listener,
		started:  false,
	}
	return reco
}

// Start connects to server and start a recognition session
func (recognizer *SpeechRecognizer) Start() error {
	recognizer.mutex.Lock()
	defer recognizer.mutex.Unlock()

	if recognizer.started {
		return fmt.Errorf("recognizer is already started")
	}
	if recognizer.VoiceID == "" {
		voiceID := uuid.New().String()
		recognizer.VoiceID = voiceID
	}
	serverURL := recognizer.buildURL(recognizer.VoiceID)
	signature := recognizer.genSignature(serverURL)

	dialer := websocket.Dialer{}
	if len(recognizer.ProxyURL) > 0 {
		proxyURL, _ := url.Parse(recognizer.ProxyURL)
		dialer.Proxy = http.ProxyURL(proxyURL)
	}

	header := http.Header(make(map[string][]string))
	urlStr := fmt.Sprintf("%s://%s&signature=%s", protocol, serverURL, url.QueryEscape(signature))
	conn, _, err := dialer.Dial(urlStr, header)
	if err != nil {
		return fmt.Errorf("voice_id: %s, error: %s", recognizer.VoiceID, err.Error())
	}
	_, data, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return fmt.Errorf("voice_id: %s, error: %s", recognizer.VoiceID, err.Error())
	}
	msg := SpeechRecognitionResponse{}
	err = json.Unmarshal(data, &msg)
	if err != nil {
		conn.Close()
		return fmt.Errorf("voice_id: %s, error: %s", recognizer.VoiceID, err.Error())
	}
	if msg.Code != 0 {
		conn.Close()
		return fmt.Errorf("voice_id: %s, code: %d, message: %s",
			recognizer.VoiceID, msg.Code, msg.Message)
	}

	recognizer.conn = conn
	go recognizer.send()
	go recognizer.receive()
	go recognizer.eventDispatch()
	recognizer.started = true

	recognizer.eventChan <- speechRecognitionEvent{
		t: eventTypeRecognitionStart,
		r: newSpeechRecognitionResponse(0, "sucess", recognizer.VoiceID,
			fmt.Sprintf("%s-RecognitionStart", recognizer.VoiceID), 0),
		err: nil,
	}
	return nil
}

// Write : write data in channel
func (recognizer *SpeechRecognizer) Write(data []byte) error {
	recognizer.mutex.Lock()
	defer recognizer.mutex.Unlock()
	if !recognizer.started {
		return fmt.Errorf("recognizer not running")
	}

	recognizer.dataChan <- data
	return nil
}

// Stop wait for the recognition process to complete
func (recognizer *SpeechRecognizer) Stop() error {
	err := recognizer.stopInternal()
	if err != nil {
		return err
	}
	return nil
}

func (recognizer *SpeechRecognizer) stopInternal() error {
	recognizer.mutex.Lock()
	defer recognizer.mutex.Unlock()
	if !recognizer.started {
		return fmt.Errorf("recognizer is not running")
	}
	close(recognizer.dataChan)
	<-recognizer.receiveEnd
	<-recognizer.sendEnd
	<-recognizer.eventEnd
	recognizer.started = false
	return nil
}

func (recognizer *SpeechRecognizer) onError(code int, message string, err error) {
	//recognizer.mutex.Lock()
	if !recognizer.started {
		return
	}

	recognizer.listener.OnFail(newSpeechRecognitionResponse(code, message, recognizer.VoiceID,
		fmt.Sprintf("%s-Error", recognizer.VoiceID), 0), err)
	/*
			recognizer.eventChan <- speechRecognitionEvent{
				t: eventTypeFail,
				r: newSpeechRecognitionResponse(code, message, recognizer.VoiceID,
					fmt.Sprintf("%s-Error", recognizer.VoiceID), 0),
				err: err,
			}
		    recognizer.mutex.Unlock()
	*/
	go recognizer.stopInternal()
}

func (recognizer *SpeechRecognizer) send() {
	defer func() {
		// handle panic
		recognizer.genRecoverFunc()()
		close(recognizer.sendEnd)
	}()
	//send data
	for data := range recognizer.dataChan {
		if err := recognizer.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
			recognizer.onError(-1, "send error", fmt.Errorf("voice_id: %s, error: %s",
				recognizer.VoiceID, err.Error()))
			return
		}
	}
	//send stop msg
	if err := recognizer.conn.WriteMessage(websocket.TextMessage, []byte("{\"type\":\"end\"}")); err != nil {
		recognizer.onError(-1, "send error", fmt.Errorf("voice_id: %s, error: %s",
			recognizer.VoiceID, err.Error()))
	}
}

func (recognizer *SpeechRecognizer) eventDispatch() {
	defer func() {
		// handle panic
		recognizer.genRecoverFunc()()
		close(recognizer.eventEnd)
	}()
	for e := range recognizer.eventChan {
		switch e.t {
		case eventTypeRecognitionStart:
			recognizer.listener.OnRecognitionStart(e.r)
		case eventTypeSentenceBegin:
			recognizer.listener.OnSentenceBegin(e.r)
		case eventTypeRecognitionResultChange:
			recognizer.listener.OnRecognitionResultChange(e.r)
		case eventTypeSentenceEnd:
			recognizer.listener.OnSentenceEnd(e.r)
		case eventTypeRecognitionComplete:
			recognizer.listener.OnRecognitionComplete(e.r)
		case eventTypeFail:
			recognizer.listener.OnFail(e.r, e.err)
		}
	}
}

func (recognizer *SpeechRecognizer) receive() {
	defer func() {
		// handle panic
		recognizer.genRecoverFunc()()
		close(recognizer.eventChan)
		close(recognizer.receiveEnd)
	}()
	index := -1
	for {
		_, data, err := recognizer.conn.ReadMessage()
		if err != nil {
			recognizer.onError(-1, "receive error", fmt.Errorf("voice_id: %s, error: %s", recognizer.VoiceID, err.Error()))
			break
		}

		//fmt.Printf("%s", data)
		msg := SpeechRecognitionResponse{}
		err = json.Unmarshal(data, &msg)
		if err != nil {
			recognizer.onError(-1, "receive error",
				fmt.Errorf("voice_id: %s, error: %s",
					recognizer.VoiceID, err.Error()))
			break
		}
		if msg.Code != 0 {
			recognizer.onError(msg.Code, msg.Message,
				fmt.Errorf("VoiceID: %s, error code %d, message: %s",
					recognizer.VoiceID, msg.Code, msg.Message))
			break
		}

		if msg.Final == 1 {
			recognizer.eventChan <- speechRecognitionEvent{
				t:   eventTypeRecognitionComplete,
				r:   &msg,
				err: nil,
			}
			break
		}

		beginOrEnd := false
		if msg.Result.Index != index || msg.Result.SliceType == 0 {
			index = msg.Result.Index
			recognizer.eventChan <- speechRecognitionEvent{
				t:   eventTypeSentenceBegin,
				r:   &msg,
				err: nil,
			}
			beginOrEnd = true
		}
		if msg.Result.SliceType == 2 {
			recognizer.eventChan <- speechRecognitionEvent{
				t:   eventTypeSentenceEnd,
				r:   &msg,
				err: nil,
			}
			beginOrEnd = true
		}
		if !beginOrEnd {
			recognizer.eventChan <- speechRecognitionEvent{
				t:   eventTypeRecognitionResultChange,
				r:   &msg,
				err: nil,
			}
		}
	}
}

func (recognizer *SpeechRecognizer) buildURL(voiceID string) string {
	var queryMap = make(map[string]string)
	queryMap["secretid"] = recognizer.Credential.SecretId
	var timestamp = time.Now().Unix()
	var timestampStr = strconv.FormatInt(timestamp, 10)
	queryMap["timestamp"] = timestampStr
	queryMap["expired"] = strconv.FormatInt(timestamp+24*60*60, 10)
	queryMap["nonce"] = timestampStr

	//params
	queryMap["engine_model_type"] = recognizer.EngineModelType
	queryMap["voice_id"] = voiceID
	queryMap["voice_format"] = strconv.FormatInt(int64(recognizer.VoiceFormat), 10)
	queryMap["needvad"] = strconv.FormatInt(int64(recognizer.NeedVad), 10)
	if recognizer.HotwordId != "" {
		queryMap["hotword_id"] = recognizer.HotwordId
	}
	if recognizer.HotwordList != "" {
		queryMap["hotword_list"] = recognizer.HotwordList
	}
	if recognizer.CustomizationId != "" {
		queryMap["customization_id"] = recognizer.CustomizationId
	}

	if recognizer.ReplaceTextId != "" {
		queryMap["replace_text_id"] = recognizer.ReplaceTextId
	}

	queryMap["filter_dirty"] = strconv.FormatInt(int64(recognizer.FilterDirty), 10)
	queryMap["filter_modal"] = strconv.FormatInt(int64(recognizer.FilterModal), 10)
	queryMap["filter_punc"] = strconv.FormatInt(int64(recognizer.FilterPunc), 10)
	queryMap["filter_empty_result"] = strconv.FormatInt(int64(recognizer.FilterEmptyResult), 10)
	queryMap["convert_num_mode"] = strconv.FormatInt(int64(recognizer.ConvertNumMode), 10)
	queryMap["word_info"] = strconv.FormatInt(int64(recognizer.WordInfo), 10)
	queryMap["reinforce_hotword"] = strconv.FormatInt(int64(recognizer.ReinforceHotword), 10)
	queryMap["max_speak_time"] = strconv.FormatInt(int64(recognizer.MaxSpeakTime), 10)
	if recognizer.VadSilenceTime > 0 {
		queryMap["vad_silence_time"] = strconv.FormatInt(int64(recognizer.VadSilenceTime), 10)
	}
	if recognizer.NoiseThreshold != 0 {
		queryMap["noise_threshold"] = strconv.FormatFloat(recognizer.NoiseThreshold, 'f', 3, 64)
	}

	var keys []string
	for k := range queryMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var queryStrBuffer bytes.Buffer
	for _, k := range keys {
		queryStrBuffer.WriteString(k)
		queryStrBuffer.WriteString("=")
		queryStrBuffer.WriteString(queryMap[k])
		queryStrBuffer.WriteString("&")
	}

	rs := []rune(queryStrBuffer.String())
	rsLen := len(rs)
	queryStr := string(rs[0 : rsLen-1])

	//gen url
	url := fmt.Sprintf("%s/asr/v2/%s?%s", host, recognizer.AppID, queryStr)
	return url
}

func (recognizer *SpeechRecognizer) genSignature(url string) string {
	hmac := hmac.New(sha1.New, []byte(recognizer.Credential.SecretKey))
	signURL := url
	hmac.Write([]byte(signURL))
	encryptedStr := hmac.Sum([]byte(nil))
	var signature = base64.StdEncoding.EncodeToString(encryptedStr)

	return signature
}

func newSpeechRecognitionResponse(code int, message string, voiceID string,
	messageID string, final uint32) *SpeechRecognitionResponse {
	return &SpeechRecognitionResponse{
		Code:      code,
		Message:   message,
		VoiceID:   voiceID,
		MessageID: messageID,
		Final:     final,
	}
}

func (recognizer *SpeechRecognizer) genRecoverFunc() func() {
	return func() {
		if r := recover(); r != nil {
			var err error
			switch r := r.(type) {
			case error:
				err = r
			default:
				err = fmt.Errorf("%v", r)
			}
			retErr := fmt.Errorf("panic error ocurred! [err: %s] [stack: %s]",
				err.Error(), string(debug.Stack()))
			recognizer.eventChan <- speechRecognitionEvent{
				t: eventTypeFail,
				r: newSpeechRecognitionResponse(-1, "panic error", recognizer.VoiceID,
					fmt.Sprintf("%s-Error", recognizer.VoiceID), 0),
				err: retErr,
			}
		}
	}
}
