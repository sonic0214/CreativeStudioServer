package video_engine

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"creative-studio-server/config"
	"creative-studio-server/pkg/logger"
)

type FFmpegProcessor struct {
	ffmpegPath  string
	ffprobePath string
}

type VideoInfo struct {
	Duration    float64 `json:"duration"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	FrameRate   float64 `json:"frame_rate"`
	Bitrate     int     `json:"bitrate"`
	Codec       string  `json:"codec"`
	Format      string  `json:"format"`
	Size        int64   `json:"size"`
	AudioCodec  string  `json:"audio_codec"`
	AudioBitrate int    `json:"audio_bitrate"`
	HasAudio    bool    `json:"has_audio"`
}

type RenderOptions struct {
	OutputFormat string  `json:"output_format"`
	Quality      string  `json:"quality"`
	Width        int     `json:"width"`
	Height       int     `json:"height"`
	FrameRate    float64 `json:"frame_rate"`
	VideoBitrate int     `json:"video_bitrate"`
	AudioBitrate int     `json:"audio_bitrate"`
	Preset       string  `json:"preset"`
	CRF          int     `json:"crf"` // Constant Rate Factor for quality
}

func NewFFmpegProcessor(cfg *config.Config) *FFmpegProcessor {
	return &FFmpegProcessor{
		ffmpegPath:  cfg.FFmpeg.FFmpegPath,
		ffprobePath: cfg.FFmpeg.FFprobePath,
	}
}

func (fp *FFmpegProcessor) GetVideoInfo(filePath string) (*VideoInfo, error) {
	// Use ffprobe to get video information
	cmd := exec.Command(fp.ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		logger.Errorf("Failed to get video info for %s: %v", filePath, err)
		return nil, fmt.Errorf("failed to analyze video: %w", err)
	}

	return fp.parseVideoInfo(output)
}

func (fp *FFmpegProcessor) parseVideoInfo(output []byte) (*VideoInfo, error) {
	var probe struct {
		Format struct {
			Duration string `json:"duration"`
			Size     string `json:"size"`
			BitRate  string `json:"bit_rate"`
		} `json:"format"`
		Streams []struct {
			CodecType    string `json:"codec_type"`
			CodecName    string `json:"codec_name"`
			Width        int    `json:"width"`
			Height       int    `json:"height"`
			RFrameRate   string `json:"r_frame_rate"`
			BitRate      string `json:"bit_rate"`
			Duration     string `json:"duration"`
			SampleRate   string `json:"sample_rate"`
			Channels     int    `json:"channels"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(output, &probe); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	info := &VideoInfo{}
	
	// Parse duration
	if duration, err := strconv.ParseFloat(probe.Format.Duration, 64); err == nil {
		info.Duration = duration
	}

	// Parse file size
	if size, err := strconv.ParseInt(probe.Format.Size, 10, 64); err == nil {
		info.Size = size
	}

	// Parse bitrate
	if bitrate, err := strconv.Atoi(probe.Format.BitRate); err == nil {
		info.Bitrate = bitrate
	}

	// Parse streams
	for _, stream := range probe.Streams {
		switch stream.CodecType {
		case "video":
			info.Width = stream.Width
			info.Height = stream.Height
			info.Codec = stream.CodecName
			
			// Parse frame rate
			if stream.RFrameRate != "" {
				parts := strings.Split(stream.RFrameRate, "/")
				if len(parts) == 2 {
					num, _ := strconv.ParseFloat(parts[0], 64)
					den, _ := strconv.ParseFloat(parts[1], 64)
					if den != 0 {
						info.FrameRate = num / den
					}
				}
			}

		case "audio":
			info.HasAudio = true
			info.AudioCodec = stream.CodecName
			if bitrate, err := strconv.Atoi(stream.BitRate); err == nil {
				info.AudioBitrate = bitrate
			}
		}
	}

	// Determine format from codec
	switch info.Codec {
	case "h264":
		info.Format = "mp4"
	case "vp9", "vp8":
		info.Format = "webm"
	case "hevc":
		info.Format = "mp4"
	default:
		info.Format = "unknown"
	}

	return info, nil
}

func (fp *FFmpegProcessor) GenerateThumbnail(inputPath, outputPath string, timeOffset float64) error {
	cmd := exec.Command(fp.ffmpegPath,
		"-i", inputPath,
		"-ss", fmt.Sprintf("%.2f", timeOffset),
		"-vframes", "1",
		"-q:v", "2",
		"-y", // Overwrite output file
		outputPath,
	)

	if err := cmd.Run(); err != nil {
		logger.Errorf("Failed to generate thumbnail: %v", err)
		return fmt.Errorf("failed to generate thumbnail: %w", err)
	}

	return nil
}

func (fp *FFmpegProcessor) ConcatenateVideos(inputPaths []string, outputPath string, options *RenderOptions) error {
	if len(inputPaths) == 0 {
		return fmt.Errorf("no input files provided")
	}

	// Create temporary concat file
	concatFile := outputPath + ".concat"
	defer os.Remove(concatFile)

	f, err := os.Create(concatFile)
	if err != nil {
		return fmt.Errorf("failed to create concat file: %w", err)
	}

	for _, path := range inputPaths {
		fmt.Fprintf(f, "file '%s'\n", path)
	}
	f.Close()

	// Build ffmpeg command
	args := []string{
		"-f", "concat",
		"-safe", "0",
		"-i", concatFile,
	}

	// Apply render options
	args = append(args, fp.buildRenderArgs(options)...)
	args = append(args, "-y", outputPath)

	cmd := exec.Command(fp.ffmpegPath, args...)
	
	if err := cmd.Run(); err != nil {
		logger.Errorf("Failed to concatenate videos: %v", err)
		return fmt.Errorf("failed to concatenate videos: %w", err)
	}

	return nil
}

func (fp *FFmpegProcessor) buildRenderArgs(options *RenderOptions) []string {
	if options == nil {
		return []string{"-c:v", "libx264", "-preset", "medium", "-crf", "23"}
	}

	var args []string

	// Video codec
	args = append(args, "-c:v", "libx264")

	// Preset
	if options.Preset != "" {
		args = append(args, "-preset", options.Preset)
	} else {
		args = append(args, "-preset", "medium")
	}

	// Quality settings
	if options.CRF > 0 {
		args = append(args, "-crf", strconv.Itoa(options.CRF))
	} else {
		// Use quality presets
		switch options.Quality {
		case "low":
			args = append(args, "-crf", "28")
		case "medium":
			args = append(args, "-crf", "23")
		case "high":
			args = append(args, "-crf", "18")
		case "ultra":
			args = append(args, "-crf", "15")
		default:
			args = append(args, "-crf", "23")
		}
	}

	// Resolution
	if options.Width > 0 && options.Height > 0 {
		args = append(args, "-s", fmt.Sprintf("%dx%d", options.Width, options.Height))
	}

	// Frame rate
	if options.FrameRate > 0 {
		args = append(args, "-r", fmt.Sprintf("%.2f", options.FrameRate))
	}

	// Bitrate
	if options.VideoBitrate > 0 {
		args = append(args, "-b:v", fmt.Sprintf("%dk", options.VideoBitrate))
	}

	// Audio settings
	args = append(args, "-c:a", "aac")
	if options.AudioBitrate > 0 {
		args = append(args, "-b:a", fmt.Sprintf("%dk", options.AudioBitrate))
	} else {
		args = append(args, "-b:a", "128k")
	}

	return args
}

func (fp *FFmpegProcessor) ExtractAudio(inputPath, outputPath string) error {
	cmd := exec.Command(fp.ffmpegPath,
		"-i", inputPath,
		"-vn", // No video
		"-acodec", "copy",
		"-y",
		outputPath,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to extract audio: %w", err)
	}

	return nil
}