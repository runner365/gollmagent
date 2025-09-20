package asrcallback

type ASRCallbackI interface {
	OnAsr2Text(text string, id string)
	OnAsrEnd(id string)
	OnAsrError(err error, id string)
}
