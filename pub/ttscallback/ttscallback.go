package ttscallback

type TtsPcmType int

const (
	PcmDelta TtsPcmType = iota + 1
	PcmDone
)

type TtsCallbackI interface {
	OnText2Pcm(pcmData []byte, id string, pcmType TtsPcmType)
	OnTtsError(err error, id string)
}
