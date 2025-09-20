package asr

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/tencentcloud/tencentcloud-speech-sdk-go/common"
	"net/http"
	"net/url"
	"runtime/debug"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// VNRecognitionListener User must impletement it. Get recognition result
type VNRecognitionListener interface {
	OnVNRecognitionStart(*VNRecognitionResponse)
	OnVNRecognitionComplete(*VNRecognitionResponse)
	OnVNFail(*VNRecognitionResponse, error)
}

// VNRecognitionResponse is the reponse of asr service
type VNRecognitionResponse struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	VoiceID   string `json:"voice_id,omitempty"`
	MessageID string `json:"message_id,omitempty"`
	Final     uint32 `json:"final,omitempty"`
	Result    uint32 `json:"result"`
}

// VNRecognizer is the entry for ASR service
type VNRecognizer struct {
	//request params
	AppID       string
	VoiceFormat int
	WaitTime    uint32 //等待时长 填0 后台默认30秒 最大60秒 单位毫秒

	Credential *common.Credential
	//listener
	listener VNRecognitionListener
	//uuid for voice
	VoiceID string

	//for proxy
	ProxyURL string

	//for websocet connection
	conn *websocket.Conn

	//send data channel
	dataChan chan []byte
	//for listener get response message
	eventChan chan VNRecognitionEvent

	//used in stop function, waiting for stop all goroutines
	sendEnd    chan int
	receiveEnd chan int
	eventEnd   chan int

	mutex   sync.Mutex
	started bool
	hasEnd  bool
}

const (
	gDefaultVoiceFormat = 1

	gProtocol = "wss"
	gHost     = "asr.cloud.tencent.com"
	gPath     = ""
)

const (
	eventTypeVNRecognitionStart    = 1
	eventTypeVNRecognitionComplete = 2
	eventTypeVNFail                = 3
)

type eventTypeVN int

type VNRecognitionEvent struct {
	t   eventTypeVN
	r   *VNRecognitionResponse
	err error
}

// NewVNRecognizer creates instance of VNRecognizer
func NewVNRecognizer(appID string, credential *common.Credential,
	listener VNRecognitionListener) *VNRecognizer {

	reco := &VNRecognizer{
		AppID:       appID,
		Credential:  credential,
		VoiceFormat: gDefaultVoiceFormat,

		dataChan:  make(chan []byte, 6400),
		eventChan: make(chan VNRecognitionEvent, 10),

		sendEnd:    make(chan int),
		receiveEnd: make(chan int),
		eventEnd:   make(chan int),

		listener: listener,
		started:  false,
	}
	return reco
}

// Start connects to server and start a recognition session
func (recognizer *VNRecognizer) Start() error {
	recognizer.mutex.Lock()
	defer recognizer.mutex.Unlock()

	if recognizer.started {
		return fmt.Errorf("recognizer is already started")
	}
	if recognizer.VoiceID == "" {
		voiceID := uuid.New().String()
		recognizer.VoiceID = voiceID
	}
	serverURL := recognizer.buildSignatureURL(recognizer.VoiceID)
	signature := recognizer.genSignature(recognizer.VoiceID)
	dialer := websocket.Dialer{}
	if len(recognizer.ProxyURL) > 0 {
		proxyURL, _ := url.Parse(recognizer.ProxyURL)
		dialer.Proxy = http.ProxyURL(proxyURL)
	}

	header := http.Header(make(map[string][]string))
	urlStr := fmt.Sprintf("%s://%s&signature=%s", gProtocol, serverURL, url.QueryEscape(signature))
	conn, _, err := dialer.Dial(urlStr, header)
	if err != nil {
		return fmt.Errorf("voice_id: %s, error: %s", recognizer.VoiceID, err.Error())
	}
	_, data, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return fmt.Errorf("voice_id: %s, error: %s", recognizer.VoiceID, err.Error())
	}
	msg := VNRecognitionResponse{}
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

	recognizer.eventChan <- VNRecognitionEvent{
		t: eventTypeVNRecognitionStart,
		r: newVNRecognitionResponse(0, "sucess", recognizer.VoiceID,
			fmt.Sprintf("%s-RecognitionStart", recognizer.VoiceID), 0),
		err: nil,
	}
	return nil
}

// Write : write data in channel
func (recognizer *VNRecognizer) Write(data []byte) (error, bool) {
	recognizer.mutex.Lock()
	defer recognizer.mutex.Unlock()
	if !recognizer.started {
		return fmt.Errorf("recognizer not running"), false
	}

	if recognizer.hasEnd {
		return nil, true
	}
	recognizer.dataChan <- data
	return nil, false
}

// Stop wait for the recognition process to complete
func (recognizer *VNRecognizer) Stop() error {
	err := recognizer.stopInternal()
	if err != nil {
		return err
	}
	return nil
}

func (recognizer *VNRecognizer) stopInternal() error {
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
	err := recognizer.conn.Close()
	if err != nil {
		return err
	}
	return nil
}

func (recognizer *VNRecognizer) onError(code int, message string, err error) {
	//recognizer.mutex.Lock()
	if !recognizer.started {
		return
	}

	recognizer.listener.OnVNFail(newVNRecognitionResponse(code, message, recognizer.VoiceID,
		fmt.Sprintf("%s-Error", recognizer.VoiceID), 0), err)
	/*
			recognizer.eventChan <- VNRecognitionEvent{
				t: eventTypeVNFail,
				r: newVNRecognitionResponse(code, message, recognizer.VoiceID,
					fmt.Sprintf("%s-Error", recognizer.VoiceID), 0),
				err: err,
			}
		    recognizer.mutex.Unlock()
	*/
	go recognizer.stopInternal()
}

func (recognizer *VNRecognizer) send() {
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

func (recognizer *VNRecognizer) eventDispatch() {
	defer func() {
		// handle panic
		recognizer.genRecoverFunc()()
		close(recognizer.eventEnd)
	}()
	for e := range recognizer.eventChan {
		switch e.t {
		case eventTypeVNRecognitionStart:
			recognizer.listener.OnVNRecognitionStart(e.r)
		case eventTypeVNRecognitionComplete:
			recognizer.listener.OnVNRecognitionComplete(e.r)
		case eventTypeVNFail:
			recognizer.listener.OnVNFail(e.r, e.err)
		}
	}
}

func (recognizer *VNRecognizer) receive() {
	defer func() {
		// handle panic
		recognizer.genRecoverFunc()()
		close(recognizer.eventChan)
		close(recognizer.receiveEnd)
	}()
	for {
		_, data, err := recognizer.conn.ReadMessage()
		if err != nil {
			recognizer.onError(-1, "receive error", fmt.Errorf("voice_id: %s, error: %s", recognizer.VoiceID, err.Error()))
			break
		}

		//fmt.Printf("%s", data)
		msg := VNRecognitionResponse{}
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
		fmt.Println("receive data:", msg)
		if msg.Final == 1 {
			recognizer.hasEnd = true
			recognizer.eventChan <- VNRecognitionEvent{
				t:   eventTypeVNRecognitionComplete,
				r:   &msg,
				err: nil,
			}
			break
		}
	}
}

func (recognizer *VNRecognizer) buildURL(voiceID string) string {
	var queryMap = make(map[string]string)
	queryMap["secretid"] = recognizer.Credential.SecretId
	var timestamp = time.Now().Unix()
	var timestampStr = strconv.FormatInt(timestamp, 10)
	queryMap["timestamp"] = timestampStr
	queryMap["expired"] = strconv.FormatInt(timestamp+24*60*60, 10)
	queryMap["nonce"] = timestampStr
	queryMap["appid"] = recognizer.AppID
	//params
	queryMap["voice_id"] = voiceID
	queryMap["voice_format"] = strconv.FormatInt(int64(recognizer.VoiceFormat), 10)
	queryMap["wait_time"] = strconv.FormatUint(uint64(recognizer.WaitTime), 10)
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
	url := fmt.Sprintf("%s/VirtualNumberTransfer?%s", gHost, queryStr)
	//url := fmt.Sprintf("%s/VirtualNumberTransfer/%s?%s", gHost, recognizer.AppID, queryStr)
	return url
}

func (recognizer *VNRecognizer) buildSignatureURL(voiceID string) string {
	var queryMap = make(map[string]string)
	queryMap["secretid"] = recognizer.Credential.SecretId
	var timestamp = time.Now().Unix()
	var timestampStr = strconv.FormatInt(timestamp, 10)
	queryMap["timestamp"] = timestampStr
	queryMap["expired"] = strconv.FormatInt(timestamp+24*60*60, 10)
	queryMap["nonce"] = timestampStr
	//params
	queryMap["voice_id"] = voiceID
	queryMap["voice_format"] = strconv.FormatInt(int64(recognizer.VoiceFormat), 10)
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

	url := fmt.Sprintf("%s/asr/virtual_number/v1/%s?%s", gHost, recognizer.AppID, queryStr)
	return url
}

func (recognizer *VNRecognizer) genSignature(voiceID string) string {
	url := recognizer.buildSignatureURL(voiceID)
	hmac := hmac.New(sha1.New, []byte(recognizer.Credential.SecretKey))
	signURL := url
	hmac.Write([]byte(signURL))
	encryptedStr := hmac.Sum([]byte(nil))
	var signature = base64.StdEncoding.EncodeToString(encryptedStr)

	return signature
}

func newVNRecognitionResponse(code int, message string, voiceID string,
	messageID string, final uint32) *VNRecognitionResponse {
	return &VNRecognitionResponse{
		Code:      code,
		Message:   message,
		VoiceID:   voiceID,
		MessageID: messageID,
		Final:     final,
	}
}

func (recognizer *VNRecognizer) genRecoverFunc() func() {
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
			recognizer.eventChan <- VNRecognitionEvent{
				t: eventTypeVNFail,
				r: newVNRecognitionResponse(-1, "panic error", recognizer.VoiceID,
					fmt.Sprintf("%s-Error", recognizer.VoiceID), 0),
				err: retErr,
			}
		}
	}
}
