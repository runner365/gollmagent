package tts

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/tencentcloud/tencentcloud-speech-sdk-go/common"
)

// SpeechSynthesisResponse SpeechSynthesisResponse
type SpeechSynthesisResponse struct {
	SessionID string
	Data      []byte
}

// SpeechSynthesisListener is the listener of
type SpeechSynthesisListener interface {
	OnMessage(*SpeechSynthesisResponse)
	OnComplete(*SpeechSynthesisResponse)
	OnCancel(*SpeechSynthesisResponse)
	OnFail(*SpeechSynthesisResponse, error)
}

// SpeechSynthesizer is the entry for TTS service
type SpeechSynthesizer struct {
	AppID      int64
	Credential *common.Credential
	VoiceType  int64
	SampleRate int64
	Codec      string

	ProxyURL string

	mutex sync.Mutex

	eventChan chan speechSynthesisEvent
	eventEnd  chan int
	sessionID string
	listener  SpeechSynthesisListener

	// 0 - idle, 1 - running, 2 - cancalled
	status      int
	statusMutex sync.Mutex
}

// TTSRequest TTSRequest
type ttsRequest struct {
	Action     string `json:"Action"`
	AppID      int64  `json:"AppId"`
	SecretID   string `json:"SecretId"`
	Timestamp  int64  `json:"Timestamp"`
	Expired    int64  `json:"Expired"`
	Text       string `json:"Text"`
	SessionID  string `json:"SessionId"`
	ModelType  int64  `json:"ModelType"`
	VoiceType  int64  `json:"VoiceType"`
	SampleRate int64  `json:"SampleRate"`
	Codec      string `json:"Codec"`
}

type ttsErrorJSONResponseError struct {
	Code    string `json:"Code"`
	Message string `json:"Message"`
}

type ttsErrorJSONResponse struct {
	RequestID string                    `json:"RequestId"`
	Error     ttsErrorJSONResponseError `json:"Error"`
}

type ttsErrorJSON struct {
	Response ttsErrorJSONResponse `json:"Response"`
}

const (
	defaultVoiceType  = 0
	defaultSampleRate = 16000
	defaultCodec      = "pcm"
	defaultAction     = "TextToStreamAudio"

	httpConnectTimeout    = 2000
	httpReadHeaderTimeout = 2000

	maxMessageSize = 10240

	protocol = "https"
	host     = "tts.cloud.tencent.com"
	path     = "/stream"
)

const (
	eventTypeMessage  = 0
	eventTypeComplete = 1
	eventTypeCancel   = 2
	eventTypeFail     = 3
)

type eventType int

type speechSynthesisEvent struct {
	t   eventType
	r   *SpeechSynthesisResponse
	err error
}

// NewSpeechSynthesizer creates instance of SpeechSynthesizer
func NewSpeechSynthesizer(appID int64, credential *common.Credential, listener SpeechSynthesisListener) *SpeechSynthesizer {
	return &SpeechSynthesizer{
		AppID:      appID,
		Credential: credential,
		VoiceType:  defaultVoiceType,
		SampleRate: defaultSampleRate,
		Codec:      defaultCodec,

		listener: listener,

		status: 0,
	}
}

// Synthesis Synthesis
func (synthesizer *SpeechSynthesizer) Synthesis(text string) error {
	synthesizer.mutex.Lock()
	defer synthesizer.mutex.Unlock()
	if synthesizer.getStatus() != 0 {
		return fmt.Errorf("synthesizer already started")
	}

	synthesizer.eventChan = make(chan speechSynthesisEvent, 10)
	synthesizer.eventEnd = make(chan int)
	go synthesizer.sendRequest(text)
	go synthesizer.eventDispatch()
	synthesizer.setStatus(1)
	return nil
}

// Cancel Cancel
func (synthesizer *SpeechSynthesizer) Cancel() error {
	synthesizer.mutex.Lock()
	defer synthesizer.mutex.Unlock()
	synthesizer.setStatus(2)
	<-synthesizer.eventEnd
	return nil
}

// Wait Wait
func (synthesizer *SpeechSynthesizer) Wait() error {
	synthesizer.mutex.Lock()
	defer synthesizer.mutex.Unlock()
	<-synthesizer.eventEnd
	return nil
}

func (synthesizer *SpeechSynthesizer) getStatus() int {
	synthesizer.statusMutex.Lock()
	defer synthesizer.statusMutex.Unlock()
	status := synthesizer.status
	return status
}

func (synthesizer *SpeechSynthesizer) setStatus(status int) {
	synthesizer.statusMutex.Lock()
	defer synthesizer.statusMutex.Unlock()
	synthesizer.status = status
}

func (synthesizer *SpeechSynthesizer) eventDispatch() {
	for e := range synthesizer.eventChan {
		switch e.t {
		case eventTypeMessage:
			synthesizer.listener.OnMessage(e.r)
		case eventTypeComplete:
			synthesizer.listener.OnComplete(e.r)
		case eventTypeCancel:
			synthesizer.listener.OnCancel(e.r)
		case eventTypeFail:
			synthesizer.listener.OnFail(e.r, e.err)
		}
	}
	synthesizer.setStatus(0)
	close(synthesizer.eventEnd)
}

func (synthesizer *SpeechSynthesizer) sendRequest(text string) {
	defer func() {
		close(synthesizer.eventChan)
	}()

	url := fmt.Sprintf("%s%s", host, path)
	var timestamp = time.Now().Unix()
	sessionID := uuid.New().String()
	req := ttsRequest{
		Action:     defaultAction,
		AppID:      synthesizer.AppID,
		SecretID:   synthesizer.Credential.SecretId,
		Timestamp:  timestamp,
		Expired:    timestamp + 24*60*60,
		Text:       text,
		SessionID:  sessionID,
		ModelType:  1,
		VoiceType:  synthesizer.VoiceType,
		SampleRate: synthesizer.SampleRate,
		Codec:      synthesizer.Codec,
	}
	signature := genSignature(url, &req, synthesizer.Credential.SecretKey)
	url = fmt.Sprintf("https://%s", url)
	postBody, err := json.Marshal(req)
	if err != nil {
		synthesizer.onError(err)
		return
	}
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(postBody))

	if err != nil {
		synthesizer.onError(err)
		return
	}
	synthesizer.sessionID = sessionID
	httpReq.Header.Add("Content-Type", "application/json; charset=UTF-8")
	httpReq.Header.Add("Authorization", signature)
	httpClient := synthesizer.createHTTPClient()
	rsp, err := httpClient.Do(httpReq)
	if err != nil {
		synthesizer.onError(err)
		return
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != 200 {
		synthesizer.onError(err)
		return
	}
	if len(rsp.Header["Content-Type"]) < 1 || rsp.Header["Content-Type"][0] != "application/octet-stream" {
		rspBody, _ := ioutil.ReadAll(rsp.Body)
		synthesizer.onError(fmt.Errorf(string(rspBody)))
		return
	}
	buffer := make([]byte, maxMessageSize, maxMessageSize)
	for {
		if synthesizer.getStatus() == 2 {
			synthesizer.onCancel()
			return
		}
		n, err := rsp.Body.Read(buffer)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			synthesizer.onError(err)
			return
		}
		if n == 0 {
			continue
		}
		copyBuf := make([]byte, n, n)
		copy(copyBuf, buffer)
		synthesizer.onMessage(copyBuf)
	}
	synthesizer.onComplete()
}

func (synthesizer *SpeechSynthesizer) onMessage(data []byte) {
	r := &SpeechSynthesisResponse{
		SessionID: synthesizer.sessionID,
		Data:      data,
	}
	event := speechSynthesisEvent{
		t:   eventTypeMessage,
		r:   r,
		err: nil,
	}
	synthesizer.eventChan <- event
}

func (synthesizer *SpeechSynthesizer) onComplete() {
	r := &SpeechSynthesisResponse{
		SessionID: synthesizer.sessionID,
	}
	event := speechSynthesisEvent{
		t:   eventTypeComplete,
		r:   r,
		err: nil,
	}
	synthesizer.eventChan <- event
}

func (synthesizer *SpeechSynthesizer) onCancel() {
	r := &SpeechSynthesisResponse{
		SessionID: synthesizer.sessionID,
	}
	synthesizer.eventChan <- speechSynthesisEvent{
		t:   eventTypeCancel,
		r:   r,
		err: nil,
	}
}

func (synthesizer *SpeechSynthesizer) onError(err error) {
	r := &SpeechSynthesisResponse{
		SessionID: synthesizer.sessionID,
	}
	synthesizer.eventChan <- speechSynthesisEvent{
		t:   eventTypeFail,
		r:   r,
		err: err,
	}
}

func (synthesizer *SpeechSynthesizer) createHTTPClient() *http.Client {
	httpTransport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: httpConnectTimeout * time.Millisecond,
		}).Dial,
		MaxIdleConns:          1,
		ResponseHeaderTimeout: httpReadHeaderTimeout * time.Millisecond,
	}
	if synthesizer.ProxyURL != "" {
		proxyURL, _ := url.Parse(synthesizer.ProxyURL)
		httpTransport.Proxy = http.ProxyURL(proxyURL)
	}
	return &http.Client{Transport: httpTransport}
}

func genSignature(url string, request *ttsRequest, secretKey string) string {
	var queryMap = make(map[string]string)
	queryMap["Action"] = request.Action
	queryMap["AppId"] = strconv.FormatInt(int64(request.AppID), 10)
	queryMap["SecretId"] = request.SecretID
	queryMap["Timestamp"] = strconv.FormatInt(int64(request.Timestamp), 10)
	queryMap["Expired"] = strconv.FormatInt(request.Expired, 10)
	queryMap["Text"] = request.Text
	queryMap["SessionId"] = request.SessionID
	queryMap["ModelType"] = strconv.FormatInt(int64(request.ModelType), 10)
	queryMap["VoiceType"] = strconv.FormatInt(int64(request.VoiceType), 10)
	queryMap["SampleRate"] = strconv.FormatInt(int64(request.SampleRate), 10)
	queryMap["Codec"] = request.Codec

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

	signURL := fmt.Sprintf("%s?%s", url, queryStr)

	hmac := hmac.New(sha1.New, []byte(secretKey))
	signURL = "POST" + signURL
	hmac.Write([]byte(signURL))
	encryptedStr := hmac.Sum([]byte(nil))
	return base64.StdEncoding.EncodeToString(encryptedStr)
}
