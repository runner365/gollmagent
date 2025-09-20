package asr

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

	"github.com/tencentcloud/tencentcloud-speech-sdk-go/common"
)

// FlashRecognitionRequest FlashRecognitionRequest
type FlashRecognitionRequest struct {
	EngineType         string `json:"engine_type"`
	VoiceFormat        string `json:"voice_format"`
	SpeakerDiarization uint32 `json:"speaker_diarization"`
	HotwordId          string `json:"hotword_id"`
	HotwordList        string `json:"hotword_list"`
	CustomizationId    string `json:"customization_id"`
	FilterDirty        int32  `json:"filter_dirty"`
	FilterModal        int32  `json:"filter_modal"`
	FilterPunc         int32  `json:"filter_punc"`
	ConvertNumMode     int32  `json:"convert_num_mode"`
	WordInfo           int32  `json:"word_info"`
	FirstChannelOnly   int32  `json:"first_channel_only"`
	ReinforceHotword   int32  `json:"reinforce_hotword"`
	SentenceMaxLength  int32  `json:"sentence_max_length"`
}

// FlashRecognitionResponse FlashRecognitionResponse
type FlashRecognitionResponse struct {
	RequestId     string                    `json:"request_id"`
	Code          int                       `json:"code"`
	Message       string                    `json:"message"`
	AudioDuration int64                     `json:"audio_duration"`
	FlashResult   []*FlashRecognitionResult `json:"flash_result,omitempty"`
}

// FlashRecognitionResult FlashRecognitionResult
type FlashRecognitionResult struct {
	Text         string                      `json:"text"`
	ChannelId    int32                       `json:"channel_id"`
	SentenceList []*FlashRecognitionSentence `json:"sentence_list,omitempty"`
}

// FlashRecognitionSentence FlashRecognitionSentence
type FlashRecognitionSentence struct {
	Text      string           `json:"text"`
	StartTime uint32           `json:"start_time"`
	EndTime   uint32           `json:"end_time"`
	SpeakerId int32            `json:"speaker_id"`
	WordList  []*FlashWordData `json:"word_list,omitempty"`
}

// FlashWordData FlashWordData
type FlashWordData struct {
	Word       string `json:"word"`
	StartTime  uint32 `json:"start_time"`
	EndTime    uint32 `json:"end_time"`
	StableFlag uint32 `json:"stable_flag"`
}

// newFlashRecognitionResponse newFlashRecognitionResponse
func newFlashRecognitionResponse(code int, message string) *FlashRecognitionResponse {
	return &FlashRecognitionResponse{
		Code:    code,
		Message: message,
	}
}

var (
	flashHost  = "asr.cloud.tencent.com"
	httpClient *http.Client

	connTimeout         = 1
	rwTimeout           = 600
	maxIdleConns        = 100
	maxIdleConnsPerHost = 2
	idleConnTimeout     = time.Duration(180) * time.Second

	//once : for once init
	once sync.Once
)

// initHttpClient init http client
func initHttpClient() {
	once.Do(func() {
		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   time.Duration(connTimeout) * time.Second,
				KeepAlive: time.Duration(rwTimeout*10) * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          maxIdleConns,
			IdleConnTimeout:       idleConnTimeout,
			TLSHandshakeTimeout:   time.Duration(connTimeout) * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
		httpClient = new(http.Client)
		httpClient.Transport = transport
		httpClient.Timeout = time.Duration(rwTimeout) * time.Second
	})
}

// FlashRecognizer is the entry for ASR flash recognizer
type FlashRecognizer struct {
	AppID string

	//for proxy
	ProxyURL string

	Credential *common.Credential
}

// NewFlashRecognizer creates instance of FlashRecognizer
func NewFlashRecognizer(appID string, credential *common.Credential) *FlashRecognizer {
	initHttpClient()
	return &FlashRecognizer{
		AppID:      appID,
		Credential: credential,
	}
}

// Recognize  Recognize
func (recognizer *FlashRecognizer) Recognize(req *FlashRecognitionRequest,
	videoData []byte) (*FlashRecognitionResponse, error) {

	signStr, reqUrl := recognizer.buildURL(req)
	signature := recognizer.genSignature(signStr)

	headers := make(map[string]string)
	headers["Host"] = flashHost
	headers["Authorization"] = signature

	if len(recognizer.ProxyURL) > 0 {
		proxyURL, _ := url.Parse(recognizer.ProxyURL)
		httpClient.Transport.(*http.Transport).Proxy = http.ProxyURL(proxyURL)
	}

	httpReq, err := http.NewRequest("POST", reqUrl, bytes.NewReader(videoData))
	if err != nil {
		return nil, fmt.Errorf("failed create http request, error: %s", err.Error())
	}
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed do request, error: %s", err.Error())
	}
	defer httpResp.Body.Close()
	respData, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed read body, error: %s", err.Error())
	}
	if httpResp.StatusCode != 200 {
		return nil, fmt.Errorf("http code not 200, respData: %s", string(respData))
	}
	resp := &FlashRecognitionResponse{}
	err = json.Unmarshal(respData, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshal, respData: %s, error: %s", respData, err.Error())
	}
	if resp.Code != 0 {
		return resp, fmt.Errorf("request_id: %s, code: %d, message: %s", resp.RequestId, resp.Code, resp.Message)
	}
	return resp, nil
}

// buildURL buildURL
func (recognizer *FlashRecognizer) buildURL(req *FlashRecognitionRequest) (string, string) {
	var queryMap = make(map[string]string)
	queryMap["secretid"] = recognizer.Credential.SecretId
	queryMap["engine_type"] = req.EngineType
	queryMap["voice_format"] = req.VoiceFormat
	queryMap["speaker_diarization"] = strconv.FormatInt(int64(req.SpeakerDiarization), 10)
	queryMap["hotword_id"] = req.HotwordId
	queryMap["hotword_list"] = req.HotwordList
	queryMap["customization_id"] = req.CustomizationId
	queryMap["filter_dirty"] = strconv.FormatInt(int64(req.FilterDirty), 10)
	queryMap["filter_modal"] = strconv.FormatInt(int64(req.FilterModal), 10)
	queryMap["filter_punc"] = strconv.FormatInt(int64(req.FilterPunc), 10)
	queryMap["convert_num_mode"] = strconv.FormatInt(int64(req.ConvertNumMode), 10)
	queryMap["word_info"] = strconv.FormatInt(int64(req.WordInfo), 10)
	queryMap["first_channel_only"] = strconv.FormatInt(int64(req.FirstChannelOnly), 10)
	queryMap["reinforce_hotword"] = strconv.FormatInt(int64(req.ReinforceHotword), 10)
	queryMap["sentence_max_length"] = strconv.FormatInt(int64(req.SentenceMaxLength), 10)
	var timestamp = time.Now().Unix()
	var timestampStr = strconv.FormatInt(timestamp, 10)
	queryMap["timestamp"] = timestampStr

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

	url := fmt.Sprintf("%s/asr/flash/v1/%s?%s", flashHost, recognizer.AppID, queryStr)
	signStr := fmt.Sprintf("POST%s", url)
	reqUrl := fmt.Sprintf("https://%s", url)
	return signStr, reqUrl
}

// genSignature genSignature
func (recognizer *FlashRecognizer) genSignature(url string) string {
	hmac := hmac.New(sha1.New, []byte(recognizer.Credential.SecretKey))
	signURL := url
	hmac.Write([]byte(signURL))
	encryptedStr := hmac.Sum([]byte(nil))
	var signature = base64.StdEncoding.EncodeToString(encryptedStr)
	return signature
}
