## 中文

### 概述
GollmAgent 是一个基于 FFmpeg 构建的智能 AI 代理，通过自然语言命令简化媒体文件处理。只需描述您想对媒体文件进行的操作，代理就会为您处理复杂的 FFmpeg 操作。

### 功能特性

#### 核心媒体处理工具
- **📹 视频转码**: 将媒体文件转换为 MP4 格式，并显示进度
- **🎵 音频提取**: 从媒体文件中提取音频并转换为 M4A 格式
- **🔗 媒体合并**: 将多个媒体文件合并为单个 MP4 文件（视频+音频）
- **🎶 音频合并**: 将多个媒体文件的音频轨道合并为单个 M4A 文件

#### 视频增强工具
- **🖼️ 图片水印**: 为视频添加图片水印，支持自定义位置
- **📝 文字水印**: 为视频添加文字水印，支持颜色和位置选项
- **📋 字幕集成**: 将 SRT 字幕文件添加到视频中

#### 视频分析工具
- **📸 帧提取**: 基于视频 I 帧生成图片
- **⏰ 时刻截图**: 在指定时间戳捕获视频帧
- **ℹ️ 系统信息**: 获取当前 FFmpeg 版本和功能

### 快速开始

#### 环境要求
- Go 1.19 或更高版本
- 已安装 FFmpeg 并可在 PATH 中访问
- （可选）编译时包含 `--enable-gpl` 和 `--enable-freetype` 的 FFmpeg（用于文字水印功能）

#### 安装
```bash
git clone https://github.com/runner365/gollmagent.git
cd gollmagent
go mod download
go build -o gollmagent .
```

#### 使用方法
```bash
./gollmagent
```

代理支持自然语言命令，例如：
- "将我的 video.avi 转换为 MP4 格式"
- "从 movie.mp4 中提取音频为 M4A"
- "给我的视频添加水印图片"
- "在 00:02:30 处截取一张图片"
- "将这些视频文件合并在一起"

### API 参考

#### 可用工具

| 工具名称 | 功能描述 | 参数 |
|---------|---------|------|
| `get_ffmpeg_version` | 获取当前 FFmpeg 版本 | 无 |
| `get_m4a_from_media_file` | 提取音频为 M4A 格式 | `input_file` |
| `transcode_with_progress` | 转换为 MP4 并显示进度 | `input_file`, `video_resolution` |
| `concat_media_files` | 合并媒体文件（视频+音频） | `input_files[]` |
| `concat_media_audio_files` | 仅合并音频轨道 | `input_files[]` |
| `image_watermark_to_video` | 添加图片水印 | `input_file`, `watermark_file`, `position` |
| `text_watermark_to_video` | 添加文字水印 | `input_file`, `watermark_text`, `position`, `color` |
| `srt_to_video` | 添加字幕 | `input_file`, `srt_file` |
| `gen_pictures_from_video` | 提取 I 帧图片 | `input_file` |
| `screenshot_at_moment` | 指定时刻截图 | `input_file`, `moment` |

#### 支持的视频分辨率
- 480p, 720p, 1080p, 1440p, 2160p (4K)

#### 水印位置
- `top-left`（左上）, `top-right`（右上）, `bottom-left`（左下）, `bottom-right`（右下）

#### 文字颜色
- 标准颜色：`white`（白色）, `black`（黑色）, `red`（红色）, `green`（绿色）, `blue`（蓝色）, `yellow`（黄色）, `cyan`（青色）, `magenta`（洋红色）

### 使用示例

```bash
# 将视频转换为 720p MP4
"将 video.avi 转换为 720p MP4 格式"

# 添加水印
"将 logo.png 作为水印添加到 video.mp4 的右上角"

# 提取音频
"从 movie.mkv 中提取音频为 M4A 文件"

# 截取图片
"在 00:01:30 时刻截取 video.mp4 的图片"
```

### 架构

项目包含几个关键组件：
- **LLM 代理**: 处理自然语言处理和工具编排
- **FFmpeg 命令**: 核心媒体处理功能
- **进度管理**: 实时操作跟踪
- **WebSocket 支持**: 实时通信
- **日志系统**: 全面的操作日志记录


### 许可证

本项目基于 MIT 许可证 - 详情请参阅 [LICENSE](LICENSE) 文件。