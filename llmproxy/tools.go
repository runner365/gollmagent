package llmproxy

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/gollmagent/ffmpegcmd"
	"github.com/gollmagent/ffmpegcmd/ffprobe"
	log "github.com/gollmagent/logging"
	"github.com/gollmagent/pub"
	"github.com/gollmagent/utils"
)

var FunctionTools []*pub.ToolDefinition
var functions map[string]pub.Function
var checkProgress pub.ProgressCallback

func InitTools() string {
	weatherTool := AddWeatherTool()
	getFfmpegVerTool := AddFfmpegTool()
	getM4aFromMediaFileTool := AddGetM4aFromMediaFileTool()
	transcodeWithProgressTool := AddTranscodeWithProgressTool()
	checkProgressTool := AddCheckProgressTool()

	FunctionTools = append(FunctionTools, getFfmpegVerTool)
	FunctionTools = append(FunctionTools, weatherTool)
	FunctionTools = append(FunctionTools, getM4aFromMediaFileTool)
	FunctionTools = append(FunctionTools, transcodeWithProgressTool)
	FunctionTools = append(FunctionTools, checkProgressTool)
	FunctionTools = append(FunctionTools, AddConcatMediaFilesTool())
	FunctionTools = append(FunctionTools, AddConcatAudioFilesTool())
	FunctionTools = append(FunctionTools, AddImageWatermark2VideoTool())
	FunctionTools = append(FunctionTools, AddTextWatermark2VideoTool())
	FunctionTools = append(FunctionTools, AddSrt2Video())
	FunctionTools = append(FunctionTools, AddGenPictureFromVideoBasedOnIFrame())
	FunctionTools = append(FunctionTools, AddScreenshotAtMomentTool())

	var desc string
	for _, tool := range FunctionTools {
		if tool.Function.Name == "get_current_weather" {
			continue
		}
		if tool.Function.Name == "check_progress" {
			continue
		}
		desc += fmt.Sprintf("工具名称: %-25s  功能: %s\n", tool.Function.Name, tool.Function.Description)
	}
	return desc
}

func GetToolFunctionByName(name string) pub.Function {
	for toolName, tool := range functions {
		if toolName == name {
			return tool
		}
	}
	return nil
}

func SetProgressCallback(cb pub.ProgressCallback) {
	checkProgress = cb
}

func AddFunctionTool(name string, desc string, parameters map[string]interface{}) *pub.ToolDefinition {
	tool := &pub.ToolDefinition{
		Type: "function",
		Function: pub.FunctionDefinition{
			Name:        name,
			Description: desc,
			Parameters:  parameters,
		},
	}

	return tool
}

func AddTranscodeWithProgressTool() *pub.ToolDefinition {
	vResDesc := "视频分辨率, 可选值: "
	for res := range ffmpegcmd.VideoResolutions {
		vResDesc += " " + res
	}
	transcodeWithProgressParams := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input_file": map[string]interface{}{
				"type":        "string",
				"description": "输入的多媒体文件路径",
			},
			"video_resolution": map[string]interface{}{
				"type":        "string",
				"description": vResDesc,
			},
		},
		"required": []string{"input_file", "video_resolution"},
	}
	tool := AddFunctionTool("transcode_with_progress", "将多媒体文件转换为mp4格式, 主要为视频文件，并显示进度", transcodeWithProgressParams)
	return tool
}

func AddImageWatermark2VideoTool() *pub.ToolDefinition {
	imageWatermark2VideoParams := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input_file": map[string]interface{}{
				"type":        "string",
				"description": "输入的多媒体文件路径",
			},
			"watermark_file": map[string]interface{}{
				"type":        "string",
				"description": "水印图片文件路径",
			},
			"position": map[string]interface{}{
				"type":        "string",
				"description": "水印位置, 可选值: top-left, top-right, bottom-left, bottom-right",
			},
		},
		"required": []string{"input_file", "watermark_file"},
	}
	tool := AddFunctionTool("image_watermark_to_video", "给视频添加图片水印", imageWatermark2VideoParams)
	return tool
}

func AddTextWatermark2VideoTool() *pub.ToolDefinition {
	colorDesc := fmt.Sprintf("水印文字颜色, 可选值: %v", ffmpegcmd.SupportedTextColor())
	textWatermark2VideoParams := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input_file": map[string]interface{}{
				"type":        "string",
				"description": "输入的多媒体文件路径",
			},
			"watermark_text": map[string]interface{}{
				"type":        "string",
				"description": "水印文字内容",
			},
			"position": map[string]interface{}{
				"type":        "string",
				"description": "水印位置, 可选值: top-left, top-right, bottom-left, bottom-right",
			},
			"color": map[string]interface{}{
				"type":        "string",
				"description": colorDesc,
			},
		},
		"required": []string{"input_file", "watermark_text"},
	}
	tool := AddFunctionTool("text_watermark_to_video", "给视频添加文字水印", textWatermark2VideoParams)
	return tool
}

func AddSrt2Video() *pub.ToolDefinition {
	srt2VideoParams := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input_file": map[string]interface{}{
				"type":        "string",
				"description": "输入的多媒体文件路径",
			},
			"srt_file": map[string]interface{}{
				"type":        "string",
				"description": "字幕文件路径",
			},
		},
		"required": []string{"input_file", "srt_file"},
	}
	tool := AddFunctionTool("srt_to_video", "给视频添加字幕", srt2VideoParams)
	return tool
}

func AddGenPictureFromVideoBasedOnIFrame() *pub.ToolDefinition {
	genPictureFromVideoBasedOnIFrameParams := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input_file": map[string]interface{}{
				"type":        "string",
				"description": "输入的视频文件路径",
			},
		},
		"required": []string{"input_file"},
	}
	tool := AddFunctionTool("gen_pictures_from_video", "基于视频的I帧生成图片", genPictureFromVideoBasedOnIFrameParams)
	return tool
}

func AddScreenshotAtMomentTool() *pub.ToolDefinition {
	screenshotAtMomentParams := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input_file": map[string]interface{}{
				"type":        "string",
				"description": "输入的视频文件路径",
			},
			"moment": map[string]interface{}{
				"type":        "string",
				"description": "截图时刻，格式为 HH:MM:SS",
			},
		},
		"required": []string{"input_file", "moment"},
	}
	tool := AddFunctionTool("screenshot_at_moment", "在指定时刻截取视频帧", screenshotAtMomentParams)
	return tool
}

func AddConcatAudioFilesTool() *pub.ToolDefinition {
	concatAudioFilesParams := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input_files": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "输入的多媒体文件路径列表",
			},
		},
		"required": []string{"input_files"},
	}
	tool := AddFunctionTool("concat_media_audio_files", "合并多个多媒体文件的音频为一个纯音频的m4a文件", concatAudioFilesParams)
	return tool
}

func AddConcatMediaFilesTool() *pub.ToolDefinition {
	concatMediaFilesParams := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input_files": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "输入的多媒体文件路径列表",
			},
		},
		"required": []string{"input_files"},
	}
	tool := AddFunctionTool("concat_media_files", "合并多个多媒体文件为一个带有视频和音频的mp4文件", concatMediaFilesParams)
	return tool
}

func AddCheckProgressTool() *pub.ToolDefinition {
	checkProgressParams := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"task_id": map[string]interface{}{
				"type":        "string",
				"description": "任务ID",
			},
		},
		"required": []string{"task_id"},
	}
	tool := AddFunctionTool("check_progress", "检查任务进度", checkProgressParams)
	return tool
}

func AddGetM4aFromMediaFileTool() *pub.ToolDefinition {
	getM4aFromMediaFileParams := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input_file": map[string]interface{}{
				"type":        "string",
				"description": "输入的多媒体文件路径",
			},
		},
		"required": []string{"input_file"},
	}
	tool := AddFunctionTool("get_m4a_from_media_file", "将多媒体文件提取音频, 转换为m4a格式", getM4aFromMediaFileParams)
	return tool
}

func AddFfmpegTool() *pub.ToolDefinition {
	getFfmpegVersionParams := map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
		"required":   []string{},
	}
	tool := AddFunctionTool("get_ffmpeg_version", "获取当前agent的ffmpeg版本", getFfmpegVersionParams)
	return tool
}

func AddWeatherTool() *pub.ToolDefinition {
	// 定义天气函数的参数（JSON Schema 格式）
	weatherParams := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"location": map[string]interface{}{
				"type":        "string",
				"description": "要查询天气的城市，例如：北京",
			},
			"unit": map[string]interface{}{
				"type":        "string",
				"enum":        []interface{}{"celsius", "fahrenheit"},
				"description": "温度单位，摄氏度或华氏度",
			},
		},
		"required": []string{"location"},
	}

	tool := AddFunctionTool("get_current_weather", "获取指定城市的当前天气信息", weatherParams)
	return tool
}

func handleToolsCall(progressCb pub.ProgressCallback, toolCalls []pub.ToolCall) (string, error) {
	for _, call := range toolCalls {
		log.Infof("工具调用: %s", call.Function.Name)
		for _, toolItem := range FunctionTools {
			if toolItem.Function.Name == call.Function.Name {
				funcHandler, exists := functions[toolItem.Function.Name]
				if !exists {
					return "", fmt.Errorf("no handler for tool: %s", toolItem.Function.Name)
				}

				// 假设参数是一个简单的 map[string]interface{}
				argsMap, ok := call.Function.Arguments.(map[string]interface{})
				if ok {
					argsMap["progress_cb"] = progressCb
					argsMap["call_id"] = call.ID
					result := funcHandler(argsMap)

					resultStr, _ := result.(string)
					return resultStr, nil
				}
				argsStr, ok := call.Function.Arguments.(string)
				if ok {
					args := make(map[string]interface{})
					json.Unmarshal([]byte(argsStr), &args)

					args["progress_cb"] = progressCb
					args["call_id"] = call.ID
					result := funcHandler(args)
					resultStr, _ := result.(string)
					return resultStr, nil
				}
				return "", fmt.Errorf("invalid arguments for tool: %s", toolItem.Function.Name)
			}
		}
	}
	return "", fmt.Errorf("no matching tool found for the call")
}

func GetWeather(args map[string]interface{}) interface{} {
	location, ok := args["location"].(string)
	if !ok {
		return "invalid location arguments for GetWeather"
	}
	unit, ok := args["unit"].(string)
	if !ok {
		return "invalid unit arguments for GetWeather"
	}
	return fmt.Sprintf("%s 当前天气（单位：%s）35°C", location, unit)
}

func GetFfmpegVersion(args map[string]interface{}) interface{} {
	ver := ffmpegcmd.GetFFmpegVersion()
	return fmt.Sprintf("ffmpeg version: %s", ver)
}

func GetM4aFromMediaFile(args map[string]interface{}) interface{} {
	inputFile, ok := args["input_file"].(string)
	if !ok {
		return "invalid input_file arguments for GetM4aFromMediaFile"
	}

	mediaInfo, err := ffprobe.GetMediaFullInfo(inputFile)
	if err != nil {
		log.Errorf("error getting media info: %v, file:%s", err, inputFile)
		return fmt.Sprintf("error getting media info: %v", err)
	}
	if !mediaInfo.HasAudio {
		return "input file has no audio stream"
	}
	output, err := ffmpegcmd.GetM4aFromMediaFile(inputFile)
	if err != nil {
		return fmt.Sprintf("error converting to m4a: %v", err)
	}
	return fmt.Sprintf("converted m4a file: %s, duration: %.02f", output, mediaInfo.Duration)
}

func TranscodeWithProgressTool(args map[string]interface{}) interface{} {
	log.Infof("TranscodeWithProgressTool called with args: %+v", args)
	inputFile, ok := args["input_file"].(string)
	if !ok {
		return "invalid input_file arguments for TranscodeWithProgressTool"
	}
	vRes, ok := args["video_resolution"].(string)
	if !ok {
		vRes = "480p"
	}
	if _, exists := ffmpegcmd.VideoResolutions[vRes]; !exists {
		log.Errorf("unsupported video_resolution: %s", vRes)
		return "unsupported video_resolution"
	}

	callId, ok := args["call_id"].(string)
	if !ok {
		return "invalid call_id arguments for TranscodeWithProgressTool"
	}
	progressObj, ok := args["progress_cb"].(pub.ProgressCallback)
	if !ok {
		return "invalid progress_cb arguments for TranscodeWithProgressTool"
	}
	out := fmt.Sprintf("%s_%s.mp4", strings.TrimSuffix(inputFile, filepath.Ext(inputFile)), vRes)
	log.Infof("Starting transcode with progress for file: %s, output:%s,callId:%s",
		inputFile, out, callId)

	mediaInfo, err := ffprobe.GetMediaFullInfo(inputFile)
	if err != nil {
		log.Errorf("error getting media info: %v, file:%s", err, inputFile)
		return fmt.Sprintf("error getting media info: %v", err)
	}
	w, h, err := ffmpegcmd.GetVideoResolution(vRes, mediaInfo.Width, mediaInfo.Height)
	if err != nil {
		log.Errorf("GetVideoResolution failed: %v, vRes: %s, mediaInfo: %+v", err, vRes, mediaInfo)
		return fmt.Sprintf("GetVideoResolution failed: %v", err)
	}

	go ffmpegcmd.TranscodeWithProgress(callId, inputFile, w, h, out, mediaInfo.Duration, progressObj)

	mediaDesc, _ := json.Marshal(&mediaInfo)

	return fmt.Sprintf("源文件信息:%s \n转码任务已启动, 需要几分钟, 输出文件: %s", string(mediaDesc), out)
}

func CheckProgressTool(args map[string]interface{}) interface{} {
	taskId, ok := args["task_id"].(string)
	if !ok {
		return "invalid taskId arguments for CheckProgressTool"
	}
	log.Infof("CheckProgressTool called with taskId: %s, checkProgress:%v", taskId, checkProgress)

	info := checkProgress.CheckProgress(taskId)
	if info == nil {
		return "No progress information found"
	}
	log.Infof("Progress info for taskId %s: %+v", taskId, info)

	msg := fmt.Sprintf("当前进度: %.2f%%, 信息: %s", info.Progress*100, info.Message)
	return msg
}

func ConcatMediaFiles(args map[string]interface{}) interface{} {
	var output string

	inputFiles, ok := args["input_files"].([]interface{})
	if !ok {
		return "invalid input_files arguments for ConcatMediaFiles"
	}
	if len(inputFiles) < 2 {
		return "need at least two input files to concat"
	}
	var inputFileStrs []string
	for _, f := range inputFiles {
		fStr, ok := f.(string)
		if ok {
			inputFileStrs = append(inputFileStrs, fStr)
		}
	}
	if len(inputFileStrs) < 2 {
		return "need at least two valid input file paths to concat"
	}

	index := time.Now().UnixMilli() % 10000
	output = fmt.Sprintf("%s_concat_%d.mp4", strings.TrimSuffix(filepath.Base(inputFileStrs[0]), filepath.Ext(inputFileStrs[0])), index)

	log.Infof("Starting to concat media files: %+v, output:%s", inputFileStrs, output)
	err := ffmpegcmd.ConcatVideosWithResize(inputFileStrs, output)
	if err != nil {
		return fmt.Sprintf("error concatenating media files: %v", err)
	}
	return output
}

func ConcatAudioFiles(args map[string]interface{}) interface{} {
	var output string

	inputFiles, ok := args["input_files"].([]interface{})
	if !ok {
		return "invalid input_files arguments for ConcatAudioFiles"
	}
	if len(inputFiles) < 2 {
		return "need at least two input files to concat"
	}
	var inputFileStrs []string
	for _, f := range inputFiles {
		fStr, ok := f.(string)
		if ok {
			inputFileStrs = append(inputFileStrs, fStr)
		}
	}
	if len(inputFileStrs) < 2 {
		return "need at least two valid input file paths to concat"
	}

	index := time.Now().UnixMilli() % 10000
	output = fmt.Sprintf("%s_audio_concat_%d.m4a", strings.TrimSuffix(filepath.Base(inputFileStrs[0]), filepath.Ext(inputFileStrs[0])), index)

	log.Infof("Starting to concat audio files: %+v, output:%s", inputFileStrs, output)
	err := ffmpegcmd.ConcatAudioOnly(inputFileStrs, output)
	if err != nil {
		return fmt.Sprintf("error concatenating audio files: %v", err)
	}
	return output
}

func ImageWatermark2Video(args map[string]interface{}) interface{} {
	inputFile, ok := args["input_file"].(string)
	if !ok {
		log.Errorf("invalid input_file arguments for ImageWatermark2Video: %+v", args)
		return "invalid input_file arguments for ImageWatermark2Video"
	}
	watermarkFile, ok := args["watermark_file"].(string)
	if !ok {
		log.Errorf("invalid watermark_file arguments for ImageWatermark2Video: %+v", args)
		return "invalid watermark_file arguments for ImageWatermark2Video"
	}
	position, ok := args["position"].(string)
	if !ok {
		position = "top-right"
	}
	videoInfo, err := ffprobe.GetMediaFullInfo(inputFile)
	if err != nil {
		log.Errorf("error getting media info: %v, file:%s", err, inputFile)
		return "error getting media info"
	}
	imgInfo, err := ffprobe.GetMediaFullInfo(watermarkFile)
	if err != nil {
		log.Errorf("error getting media info: %v, file:%s", err, watermarkFile)
		return "error getting media info"
	}

	x, y, err := ffmpegcmd.GetPosition(position, videoInfo.Width, videoInfo.Height, imgInfo.Width, imgInfo.Height)
	if err != nil {
		log.Errorf("error getting position: %v", err)
		return "error getting position"
	}
	output := fmt.Sprintf("%s_watermarked.mp4", strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile)))

	log.Infof("Starting to add image watermark to video: %s, watermark:%s, position:%s, output:%s",
		inputFile, watermarkFile, position, output)
	err = ffmpegcmd.ImageWatermark2Video(inputFile, watermarkFile, x, y, output)
	if err != nil {
		return fmt.Sprintf("error adding image watermark to video: %v", err)
	}
	return output
}

func TextWatermark2Video(args map[string]interface{}) interface{} {
	inputFile, ok := args["input_file"].(string)
	if !ok {
		log.Errorf("invalid input_file arguments for TextWatermark2Video: %+v", args)
		return "invalid input_file arguments for TextWatermark2Video"
	}
	watermarkText, ok := args["watermark_text"].(string)
	if !ok {
		log.Errorf("invalid watermark_text arguments for TextWatermark2Video: %+v", args)
		return "invalid watermark_text arguments for TextWatermark2Video"
	}
	position, ok := args["position"].(string)
	if !ok {
		position = "top-right"
	}
	colorString, ok := args["color"].(string)
	if !ok {
		colorString = "white"
	}

	configs, err := ffmpegcmd.GetFFmpegConfig()
	if err != nil {
		log.Errorf("error getting ffmpeg config: %v", err)
		return "error getting ffmpeg config"
	}
	// check if ffmpeg is compiled with --enable-gpl --enable-freetype
	if !strings.Contains(strings.Join(configs, " "), "--enable-gpl") || !strings.Contains(strings.Join(configs, " "), "--enable-freetype") {
		log.Errorf("ffmpeg is not compiled with --enable-gpl --enable-freetype, cannot add text watermark")
		return "ffmpeg is not compiled with --enable-gpl --enable-freetype, cannot add text watermark"
	}

	videoInfo, err := ffprobe.GetMediaFullInfo(inputFile)
	if err != nil {
		log.Errorf("error getting media info: %v, file:%s", err, inputFile)
		return fmt.Errorf("error getting media info: %v", err)
	}
	x, y, err := ffmpegcmd.GetTextPosition(position, videoInfo.Width, videoInfo.Height, 24)
	if err != nil {
		log.Errorf("error getting text position: %v", err)
		return fmt.Errorf("error getting text position: %v", err)
	}
	output := fmt.Sprintf("%s_text_watermarked.mp4", strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile)))

	log.Infof("Starting to add text watermark to video: %s, text:%s, position:%s, color:%s, output:%s",
		inputFile, watermarkText, position, colorString, output)
	err = ffmpegcmd.TextWatermark2Video(inputFile, watermarkText, x, y, colorString, output)
	if err != nil {
		log.Errorf("error adding text watermark to video: %v", err)
		return fmt.Sprintf("error adding text watermark to video: %v", err)
	}
	return output
}

func Srt2Video(args map[string]interface{}) interface{} {
	inputFile, ok := args["input_file"].(string)
	if !ok {
		log.Errorf("invalid input_file arguments for Srt2Video: %+v", args)
		return "invalid input_file arguments for Srt2Video"
	}
	srtFile, ok := args["srt_file"].(string)
	if !ok {
		log.Errorf("invalid srt_file arguments for Srt2Video: %+v", args)
		return "invalid srt_file arguments for Srt2Video"
	}
	output := fmt.Sprintf("%s_srt.mp4", strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile)))

	log.Infof("Starting to add srt to video: %s, srt:%s, output:%s",
		inputFile, srtFile, output)
	err := ffmpegcmd.Srt2Video(inputFile, srtFile, output)
	if err != nil {
		log.Errorf("error adding srt to video: %v", err)
		return fmt.Sprintf("error adding srt to video: %v", err)
	}
	return output
}

func GenPicturesFromVideoBaseOnIFrame(args map[string]interface{}) interface{} {
	inputFile, ok := args["input_file"].(string)
	if !ok {
		log.Errorf("invalid input_file arguments %+v", args)
		return "invalid input_file arguments"
	}
	outputDir := strings.TrimSuffix(inputFile, filepath.Ext(inputFile)) + "_pics"

	err := utils.EnsureDir(outputDir)
	if err != nil {
		log.Errorf("error ensuring output directory: %v", err)
		return fmt.Sprintf("error ensuring output directory: %v", err)
	}

	log.Infof("Starting to gen pictures from video based on I-frame: %s, outputDir:%s",
		inputFile, outputDir)
	err = ffmpegcmd.GenPictureFromVideoBaseOnIframe(inputFile, outputDir)
	if err != nil {
		log.Errorf("error generating pictures from video: %v", err)
		return fmt.Sprintf("error generating pictures from video: %v", err)
	}
	return outputDir
}

func ScreenshotOnePictureAtMoment(args map[string]interface{}) interface{} {
	inputFile, ok := args["input_file"].(string)
	if !ok {
		log.Errorf("invalid input_file arguments %+v", args)
		return "invalid input_file arguments"
	}
	moment, ok := args["moment"].(string)
	if !ok {
		moment = "00:00:01"
	}
	if !ffmpegcmd.IsValidFFmpegTimeFormat(moment) {
		log.Errorf("invalid moment format: %s", moment)
		return "invalid moment format, should be HH:MM:SS"
	}
	hours, minutes, seconds, err := ffmpegcmd.GetTimeFromTimeFormat(moment)
	if err != nil {
		log.Errorf("error getting time from moment: %v", err)
		return fmt.Sprintf("error getting time from moment: %v", err)
	}

	output := fmt.Sprintf("%s_screenshot_%02d_%02d_%02d.png", strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile)), hours, minutes, seconds)

	log.Infof("Starting to screenshot one picture at moment: %s, moment:%s, output:%s",
		inputFile, moment, output)
	err = ffmpegcmd.ScreenshotOnePictureAtMoment(inputFile, moment, output)
	if err != nil {
		log.Errorf("error screenshotting picture from video: %v", err)
		return fmt.Sprintf("error screenshotting picture from video: %v", err)
	}
	return output
}

func CreateFunctionToolsHandler() {
	functions = make(map[string]pub.Function)
	functions["get_current_weather"] = GetWeather
	functions["get_ffmpeg_version"] = GetFfmpegVersion
	functions["get_m4a_from_media_file"] = GetM4aFromMediaFile
	functions["transcode_with_progress"] = TranscodeWithProgressTool
	functions["check_progress"] = CheckProgressTool
	functions["concat_media_files"] = ConcatMediaFiles
	functions["concat_media_audio_files"] = ConcatAudioFiles
	functions["image_watermark_to_video"] = ImageWatermark2Video
	functions["text_watermark_to_video"] = TextWatermark2Video
	functions["srt_to_video"] = Srt2Video
	functions["gen_pictures_from_video"] = GenPicturesFromVideoBaseOnIFrame
	functions["screenshot_at_moment"] = ScreenshotOnePictureAtMoment
}
