package pub

type ProgressInfo struct {
	Progress float32 // 0.0 - 1.0
	Message  string  // 进度描述信息
	Done     bool    // 是否完成
	Ms       uint64
}

type ProgressCallback interface {
	OnProgress(info *ProgressInfo, id string)
	CheckProgress(id string) *ProgressInfo
}
