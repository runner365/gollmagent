package tts

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"net/url"
	"runtime/debug"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tencentcloud/tencentcloud-speech-sdk-go/common"
)

// SpeechWsSynthesisResponse response
type SpeechWsSynthesisResponse struct {
	SessionId string             `json:"session_id"` //音频流唯一 id，由客户端在握手阶段生成并赋值在调用参数中
	RequestId string             `json:"request_id"` //音频流唯一 id，由服务端在握手阶段自动生成
	MessageId string             `json:"message_id"` //本 message 唯一 id
	Code      int                `json:"code"`       //状态码，0代表正常，非0值表示发生错误
	Message   string             `json:"message"`    //错误说明，发生错误时显示这个错误发生的具体原因，随着业务发展或体验优化，此文本可能会经常保持变更或更新
	Result    SynthesisSubtitles `json:"result"`     //最新语音合成文本结果
	Final     int                `json:"final"`      //该字段返回1时表示文本全部合成结束，客户端收到后需主动关闭 websocket 连接
}

func (s *SpeechWsSynthesisResponse) ToString() string {
	d, _ := json.Marshal(s)
	return string(d)
}

// SynthesisSubtitles subtitles
type SynthesisSubtitles struct {
	Subtitles []SynthesisSubtitle `json:"subtitles"`
}

// SynthesisSubtitle  Subtitle
type SynthesisSubtitle struct {
	Text       string
	Phoneme    string
	BeginTime  int64
	EndTime    int64
	BeginIndex int
	EndIndex   int
}

// SpeechWsSynthesizer is the entry for TTS websocket service
type SpeechWsSynthesizer struct {
	Credential       *common.Credential
	action           string  `json:"Action"`
	AppID            int64   `json:"AppId"`
	Timestamp        int64   `json:"Timestamp"`
	Expired          int64   `json:"Expired"`
	SessionId        string  `json:"SessionId"`
	Text             string  `json:"Text"`
	ModelType        int64   `json:"ModelType"`
	VoiceType        int64   `json:"VoiceType"`
	SampleRate       int64   `json:"SampleRate"`
	Codec            string  `json:"Codec"`
	Speed            float64 `json:"Speed"`
	Volume           float64 `json:"Volume"`
	EnableSubtitle   bool    `json:"EnableSubtitle"`
	EmotionCategory  string  `json:"EmotionCategory"`
	EmotionIntensity int64   `json:"EmotionIntensity"`
	SegmentRate      int64   `json:"SegmentRate"`
	FastVoiceType    string  `json:"FastVoiceType"`
	ExtParam         map[string]string

	ProxyURL    string
	mutex       sync.Mutex
	receiveEnd  chan int
	eventChan   chan speechWsSynthesisEvent
	eventEnd    chan int
	listener    SpeechWsSynthesisListener
	status      int
	statusMutex sync.Mutex
	conn        *websocket.Conn //for websocet connection
	started     bool

	Debug     bool //是否debug
	DebugFunc func(message string)
}

// SpeechWsSynthesisListener is the listener of
type SpeechWsSynthesisListener interface {
	OnSynthesisStart(*SpeechWsSynthesisResponse)
	OnSynthesisEnd(*SpeechWsSynthesisResponse)
	OnAudioResult(data []byte)
	OnTextResult(*SpeechWsSynthesisResponse)
	OnSynthesisFail(*SpeechWsSynthesisResponse, error)
}

const (
	defaultWsVoiceType  = 0
	defaultWsSampleRate = 16000
	defaultWsCodec      = "pcm"
	defaultWsAction     = "TextToStreamAudioWS"
	wsConnectTimeout    = 2000
	wsReadHeaderTimeout = 2000
	maxWsMessageSize    = 10240
	wsProtocol          = "wss"
	wsHost              = "tts.cloud.tencent.com"
	wsPath              = "/stream_ws"
)

const (
	eventTypeWsStart = iota
	eventTypeWsEnd
	eventTypeWsAudioResult
	eventTypeWsTextResult
	eventTypeWsFail
)

type eventWsType int

type speechWsSynthesisEvent struct {
	t   eventWsType
	r   *SpeechWsSynthesisResponse
	d   []byte
	err error
}

// NewSpeechWsSynthesizer creates instance of SpeechWsSynthesizer
func NewSpeechWsSynthesizer(appID int64, credential *common.Credential, listener SpeechWsSynthesisListener) *SpeechWsSynthesizer {
	return &SpeechWsSynthesizer{
		AppID:      appID,
		Credential: credential,
		action:     defaultWsAction,
		VoiceType:  defaultWsVoiceType,
		SampleRate: defaultWsSampleRate,
		Codec:      defaultWsCodec,
		listener:   listener,
		status:     0,
		receiveEnd: make(chan int),
		eventChan:  make(chan speechWsSynthesisEvent, 10),
		eventEnd:   make(chan int),
	}
}

// Synthesis Start connects to server and start a synthesizer session
func (synthesizer *SpeechWsSynthesizer) Synthesis() error {
	synthesizer.mutex.Lock()
	defer synthesizer.mutex.Unlock()

	if synthesizer.started {
		return fmt.Errorf("synthesizer is already started")
	}
	if synthesizer.SessionId == "" {
		SessionId := uuid.New().String()
		synthesizer.SessionId = SessionId
	}
	var timestamp = time.Now().Unix()
	synthesizer.Timestamp = timestamp
	synthesizer.Expired = timestamp + 24*60*60
	serverURL := synthesizer.buildURL(false)
	signature := synthesizer.genWsSignature(serverURL, synthesizer.Credential.SecretKey)
	if synthesizer.Debug && synthesizer.DebugFunc != nil {
		logMsg := fmt.Sprintf("serverURL:%s , signature:%s", serverURL, signature)
		synthesizer.DebugFunc(logMsg)
	}
	dialer := websocket.Dialer{}
	if len(synthesizer.ProxyURL) > 0 {
		proxyURL, _ := url.Parse(synthesizer.ProxyURL)
		dialer.Proxy = http.ProxyURL(proxyURL)
	}
	serverURL = synthesizer.buildURL(true)
	header := http.Header(make(map[string][]string))
	urlStr := fmt.Sprintf("%s://%s&Signature=%s", wsProtocol, serverURL, url.QueryEscape(signature))
	if synthesizer.Debug && synthesizer.DebugFunc != nil {
		logMsg := fmt.Sprintf("urlStr:%s ", urlStr)
		synthesizer.DebugFunc(logMsg)
	}
	conn, _, err := dialer.Dial(urlStr, header)
	if err != nil {
		return fmt.Errorf("session_id: %s, error: %s", synthesizer.SessionId, err.Error())
	}
	_, data, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return fmt.Errorf("session_id: %s, error: %s", synthesizer.SessionId, err.Error())
	}
	msg := SpeechWsSynthesisResponse{}
	err = json.Unmarshal(data, &msg)
	if err != nil {
		conn.Close()
		return fmt.Errorf("session_id: %s, error: %s", synthesizer.SessionId, err.Error())
	}
	if msg.Code != 0 {
		conn.Close()
		return fmt.Errorf("session_id: %s, code: %d, message: %s",
			synthesizer.SessionId, msg.Code, msg.Message)
	}
	msg.SessionId = synthesizer.SessionId
	synthesizer.conn = conn
	go synthesizer.receive()
	go synthesizer.eventDispatch()
	synthesizer.started = true
	synthesizer.setStatus(eventTypeWsStart)
	synthesizer.eventChan <- speechWsSynthesisEvent{
		t:   eventTypeWsStart,
		r:   &msg,
		err: nil,
	}
	return nil
}

func (synthesizer *SpeechWsSynthesizer) receive() {
	defer func() {
		// handle panic
		synthesizer.genRecoverFunc()()
		close(synthesizer.eventChan)
		close(synthesizer.receiveEnd)
	}()
	for {
		optCode, data, err := synthesizer.conn.ReadMessage()
		if err != nil {
			synthesizer.onError(fmt.Errorf("SessionId: %s, error: %s", synthesizer.SessionId, err.Error()))
			break
		}
		if optCode == websocket.BinaryMessage {
			if synthesizer.Debug && synthesizer.DebugFunc != nil {
				synthesizer.DebugFunc(fmt.Sprintf("[%s] receive binary message size: %d", synthesizer.SessionId, len(data)))
			}
			msg := SpeechWsSynthesisResponse{SessionId: synthesizer.SessionId}
			synthesizer.eventChan <- speechWsSynthesisEvent{
				t:   eventTypeWsAudioResult,
				r:   &msg,
				d:   data,
				err: nil,
			}
		}
		if optCode == websocket.TextMessage {
			if synthesizer.Debug && synthesizer.DebugFunc != nil {
				synthesizer.DebugFunc(fmt.Sprintf("[%s]  receive text message: %s", synthesizer.SessionId, string(data)))
			}
			msg := SpeechWsSynthesisResponse{}
			err = json.Unmarshal(data, &msg)
			if err != nil {
				synthesizer.onError(fmt.Errorf("SessionId: %s, error: %s",
					synthesizer.SessionId, err.Error()))
				break
			}
			msg.SessionId = synthesizer.SessionId
			if msg.Code != 0 {
				synthesizer.onErrorResp(msg, fmt.Errorf("VoiceID: %s, error code %d, message: %s",
					synthesizer.SessionId, msg.Code, msg.Message))
				break
			}
			if msg.Final == 1 {
				synthesizer.setStatus(eventTypeWsEnd)
				synthesizer.closeConn()
				synthesizer.eventChan <- speechWsSynthesisEvent{
					t:   eventTypeWsEnd,
					r:   &msg,
					err: nil,
				}
				break
			}
			synthesizer.eventChan <- speechWsSynthesisEvent{
				t:   eventTypeWsTextResult,
				r:   &msg,
				err: nil,
			}
		}
	}
}

func (synthesizer *SpeechWsSynthesizer) eventDispatch() {
	defer func() {
		// handle panic
		synthesizer.genRecoverFunc()()
		close(synthesizer.eventEnd)
	}()
	for e := range synthesizer.eventChan {
		switch e.t {
		case eventTypeWsStart:
			synthesizer.listener.OnSynthesisStart(e.r)
		case eventTypeWsEnd:
			synthesizer.listener.OnSynthesisEnd(e.r)
		case eventTypeWsAudioResult:
			synthesizer.listener.OnAudioResult(e.d)
		case eventTypeWsTextResult:
			synthesizer.listener.OnTextResult(e.r)
		case eventTypeWsFail:
			synthesizer.listener.OnSynthesisFail(e.r, e.err)
		}
	}
}

// Wait Wait
func (synthesizer *SpeechWsSynthesizer) Wait() error {
	synthesizer.mutex.Lock()
	defer synthesizer.mutex.Unlock()
	<-synthesizer.eventEnd
	<-synthesizer.receiveEnd
	return nil
}

func (synthesizer *SpeechWsSynthesizer) getStatus() int {
	synthesizer.statusMutex.Lock()
	defer synthesizer.statusMutex.Unlock()
	status := synthesizer.status
	return status
}

func (synthesizer *SpeechWsSynthesizer) setStatus(status int) {
	synthesizer.statusMutex.Lock()
	defer synthesizer.statusMutex.Unlock()
	synthesizer.status = status
}

func (synthesizer *SpeechWsSynthesizer) onError(err error) {
	r := &SpeechWsSynthesisResponse{
		SessionId: synthesizer.SessionId,
	}
	synthesizer.closeConn()
	synthesizer.eventChan <- speechWsSynthesisEvent{
		t:   eventTypeWsFail,
		r:   r,
		err: err,
	}
}

func (synthesizer *SpeechWsSynthesizer) onErrorResp(resp SpeechWsSynthesisResponse, err error) {
	synthesizer.closeConn()
	synthesizer.eventChan <- speechWsSynthesisEvent{
		t:   eventTypeWsFail,
		r:   &resp,
		err: err,
	}
}

func (synthesizer *SpeechWsSynthesizer) buildURL(escape bool) string {
	var queryMap = make(map[string]string)
	queryMap["Action"] = synthesizer.action
	queryMap["AppId"] = strconv.FormatInt(synthesizer.AppID, 10)
	queryMap["SecretId"] = synthesizer.Credential.SecretId
	queryMap["Timestamp"] = strconv.FormatInt(synthesizer.Timestamp, 10)
	queryMap["Expired"] = strconv.FormatInt(synthesizer.Expired, 10)
	if escape {
		//url escapes the string so it can be safely placed
		queryMap["Text"] = url.QueryEscape(synthesizer.Text)
	} else {
		queryMap["Text"] = synthesizer.Text
	}
	queryMap["FastVoiceType"] = synthesizer.FastVoiceType
	queryMap["SessionId"] = synthesizer.SessionId
	queryMap["ModelType"] = strconv.FormatInt(synthesizer.ModelType, 10)
	queryMap["VoiceType"] = strconv.FormatInt(synthesizer.VoiceType, 10)
	queryMap["SampleRate"] = strconv.FormatInt(synthesizer.SampleRate, 10)
	queryMap["Speed"] = strconv.FormatFloat(synthesizer.Speed, 'g', -1, 64)
	queryMap["Volume"] = strconv.FormatFloat(synthesizer.Volume, 'g', -1, 64)
	queryMap["Codec"] = synthesizer.Codec
	queryMap["EnableSubtitle"] = strconv.FormatBool(synthesizer.EnableSubtitle)
	queryMap["EmotionCategory"] = synthesizer.EmotionCategory
	queryMap["EmotionIntensity"] = strconv.FormatInt(synthesizer.EmotionIntensity, 10)
	queryMap["SegmentRate"] = strconv.FormatInt(synthesizer.SegmentRate, 10)
	for k, v := range synthesizer.ExtParam {
		queryMap[k] = v
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
	serverURL := fmt.Sprintf("%s%s", wsHost, wsPath)
	signURL := fmt.Sprintf("%s?%s", serverURL, queryStr)
	return signURL
}

func (synthesizer *SpeechWsSynthesizer) genWsSignature(signURL string, secretKey string) string {
	hmac := hmac.New(sha1.New, []byte(secretKey))
	signURL = "GET" + signURL
	hmac.Write([]byte(signURL))
	encryptedStr := hmac.Sum([]byte(nil))
	return base64.StdEncoding.EncodeToString(encryptedStr)
}

func (synthesizer *SpeechWsSynthesizer) genRecoverFunc() func() {
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
			msg := SpeechWsSynthesisResponse{
				SessionId: synthesizer.SessionId,
			}
			synthesizer.eventChan <- speechWsSynthesisEvent{
				t:   eventTypeWsFail,
				r:   &msg,
				err: retErr,
			}
		}
	}
}

// CloseConn close connection
func (synthesizer *SpeechWsSynthesizer) CloseConn() {
	synthesizer.closeConn()
}

func (synthesizer *SpeechWsSynthesizer) closeConn() {
	err := synthesizer.conn.Close()
	if err != nil && synthesizer.Debug && synthesizer.DebugFunc != nil {
		synthesizer.DebugFunc(fmt.Sprintf("%s %s", time.Now().String(), err.Error()))
	}
}
