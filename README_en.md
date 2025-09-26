# GollmAgent - FFmpeg AI Agent

[English](#english) | [中文](README.md)

## English

### Overview
GollmAgent is an intelligent AI agent built on top of FFmpeg that simplifies media file processing through natural language commands. Simply describe what you want to do with your media files, and the agent will handle the complex FFmpeg operations for you.

### Features

#### Core Media Processing Tools
- **📹 Video Transcoding**: Convert media files to MP4 format with progress tracking
- **🎵 Audio Extraction**: Extract audio from media files and convert to M4A format
- **🔗 Media Concatenation**: Merge multiple media files into a single MP4 (video + audio)
- **🎶 Audio Concatenation**: Merge audio tracks from multiple media files into a single M4A file

#### Video Enhancement Tools
- **🖼️ Image Watermarks**: Add image watermarks to videos with customizable positioning
- **📝 Text Watermarks**: Add text watermarks to videos with color and position options
- **📋 Subtitle Integration**: Add SRT subtitle files to videos

#### Video Analysis Tools
- **📸 Frame Extraction**: Generate images based on video I-frames
- **⏰ Moment Screenshots**: Capture video frames at specific timestamps
- **ℹ️ System Information**: Get current FFmpeg version and capabilities

### Quick Start

#### Prerequisites
- Go 1.19 or higher
- FFmpeg installed and accessible in PATH
- (Optional) FFmpeg compiled with `--enable-gpl` and `--enable-freetype` for text watermarks

#### Installation
```bash
git clone https://github.com/runner365/gollmagent.git
cd gollmagent
go mod download
go build -o gollmagent .
```

#### Usage
```bash
./gollmagent
```

The agent supports natural language commands like:
- "Convert my video.avi to MP4 format"
- "Extract audio from movie.mp4 as M4A"
- "Add a watermark image to my video"
- "Take a screenshot at 00:02:30"
- "Merge these video files together"

### API Reference

#### Available Tools

| Tool Name | Description | Parameters |
|-----------|-------------|------------|
| `get_ffmpeg_version` | Get current FFmpeg version | None |
| `get_m4a_from_media_file` | Extract audio to M4A format | `input_file` |
| `transcode_with_progress` | Convert to MP4 with progress | `input_file`, `video_resolution` |
| `concat_media_files` | Merge media files (video+audio) | `input_files[]` |
| `concat_media_audio_files` | Merge audio tracks only | `input_files[]` |
| `image_watermark_to_video` | Add image watermark | `input_file`, `watermark_file`, `position` |
| `text_watermark_to_video` | Add text watermark | `input_file`, `watermark_text`, `position`, `color` |
| `srt_to_video` | Add subtitles | `input_file`, `srt_file` |
| `gen_pictures_from_video` | Extract I-frame images | `input_file` |
| `screenshot_at_moment` | Screenshot at timestamp | `input_file`, `moment` |

#### Supported Video Resolutions
- 480p, 720p, 1080p, 1440p, 2160p (4K)

#### Watermark Positions
- `top-left`, `top-right`, `bottom-left`, `bottom-right`

#### Text Colors
- Standard colors: `white`, `black`, `red`, `green`, `blue`, `yellow`, `cyan`, `magenta`

### Examples

```bash
# Convert video to 720p MP4
"Convert video.avi to 720p MP4 format"

# Add watermark
"Add logo.png as watermark to video.mp4 in top-right corner"

# Extract audio
"Extract audio from movie.mkv as M4A file"

# Take screenshot
"Take a screenshot of video.mp4 at 00:01:30"
```

### Architecture

The project consists of several key components:
- **LLM Proxy**: Handles natural language processing and tool orchestration
- **FFmpeg Commands**: Core media processing functionality
- **Progress Management**: Real-time operation tracking
- **WebSocket Support**: For real-time communication
- **Logging System**: Comprehensive operation logging


### License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

