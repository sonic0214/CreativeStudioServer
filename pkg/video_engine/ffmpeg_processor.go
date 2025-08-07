package video_engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	Filters      []VideoFilter `json:"filters"`
}

type VideoFilter struct {
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"parameters"`
}

type RenderProgress struct {
	Frame     int     `json:"frame"`
	FPS       float64 `json:"fps"`
	Bitrate   string  `json:"bitrate"`
	TotalSize int64   `json:"total_size"`
	Time      string  `json:"time"`
	Speed     float64 `json:"speed"`
	Progress  float64 `json:"progress"`
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

func (fp *FFmpegProcessor) RenderComposition(composition *CompositionResult, outputPath string, options *RenderOptions, progressCallback func(*RenderProgress)) error {
	// Create complex filter for timeline rendering
	filterComplex := fp.buildFilterComplex(composition)
	
	// Build input arguments
	var args []string
	inputClips := make(map[uint]string)
	
	// Add input files
	for _, clip := range composition.SelectedClips {
		if path, exists := inputClips[clip.ClipID]; !exists {
			// In a real implementation, you'd get the actual file path from the database
			path = fmt.Sprintf("/path/to/clip/%d.mp4", clip.ClipID)
			inputClips[clip.ClipID] = path
			args = append(args, "-i", path)
		}
	}

	// Add filter complex
	if filterComplex != "" {
		args = append(args, "-filter_complex", filterComplex)
	}

	// Add render options
	args = append(args, fp.buildRenderArgs(options)...)
	
	// Progress reporting
	if progressCallback != nil {
		args = append(args, "-progress", "pipe:1")
	}

	args = append(args, "-y", outputPath)

	cmd := exec.Command(fp.ffmpegPath, args...)
	
	if progressCallback != nil {
		return fp.runWithProgress(cmd, composition.TotalDuration, progressCallback)
	}

	if err := cmd.Run(); err != nil {
		logger.Errorf("Failed to render composition: %v", err)
		return fmt.Errorf("failed to render composition: %w", err)
	}

	return nil
}

func (fp *FFmpegProcessor) buildFilterComplex(composition *CompositionResult) string {
	if len(composition.SelectedClips) == 0 {
		return ""
	}

	var filters []string
	currentTime := 0.0

	for i, clip := range composition.SelectedClips {
		// Trim clip to specified duration
		trimFilter := fmt.Sprintf("[%d:v]trim=start=%.2f:duration=%.2f,setpts=PTS-STARTPTS[v%d]", 
			i, clip.StartTime, clip.Duration, i)
		filters = append(filters, trimFilter)

		// Audio trim if needed
		audioTrimFilter := fmt.Sprintf("[%d:a]atrim=start=%.2f:duration=%.2f,asetpts=PTS-STARTPTS[a%d]", 
			i, clip.StartTime, clip.Duration, i)
		filters = append(filters, audioTrimFilter)
	}

	// Concatenate all clips
	if len(composition.SelectedClips) > 1 {
		var videoInputs, audioInputs string
		for i := range composition.SelectedClips {
			videoInputs += fmt.Sprintf("[v%d]", i)
			audioInputs += fmt.Sprintf("[a%d]", i)
		}

		concatFilter := fmt.Sprintf("%s concat=n=%d:v=1:a=1[outv][outa]", 
			videoInputs, len(composition.SelectedClips))
		filters = append(filters, concatFilter)
	}

	return strings.Join(filters, ";")
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

	// Apply filters
	if len(options.Filters) > 0 {
		filterStr := fp.buildVideoFilters(options.Filters)
		if filterStr != "" {
			args = append(args, "-vf", filterStr)
		}
	}

	return args
}

func (fp *FFmpegProcessor) buildVideoFilters(filters []VideoFilter) string {
	var filterStrings []string

	for _, filter := range filters {
		filterStr := filter.Name
		if len(filter.Parameters) > 0 {
			var params []string
			for key, value := range filter.Parameters {
				params = append(params, fmt.Sprintf("%s=%v", key, value))
			}
			filterStr += "=" + strings.Join(params, ":")
		}
		filterStrings = append(filterStrings, filterStr)
	}

	return strings.Join(filterStrings, ",")
}

func (fp *FFmpegProcessor) runWithProgress(cmd *exec.Cmd, totalDuration float64, progressCallback func(*RenderProgress)) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		scanner := NewProgressScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if progress := fp.parseProgress(line, totalDuration); progress != nil {
				progressCallback(progress)
			}
		}
	}()

	return cmd.Wait()
}

func (fp *FFmpegProcessor) parseProgress(line string, totalDuration float64) *RenderProgress {
	// Parse FFmpeg progress output
	// Format: frame=  123 fps= 25 q=28.0 size=     456kB time=00:00:05.12 bitrate=1234.5kbits/s speed=   1x
	
	progress := &RenderProgress{}
	parts := strings.Fields(line)

	for _, part := range parts {
		if strings.HasPrefix(part, "frame=") {
			if frame, err := strconv.Atoi(strings.TrimPrefix(part, "frame=")); err == nil {
				progress.Frame = frame
			}
		} else if strings.HasPrefix(part, "fps=") {
			if fps, err := strconv.ParseFloat(strings.TrimPrefix(part, "fps="), 64); err == nil {
				progress.FPS = fps
			}
		} else if strings.HasPrefix(part, "time=") {
			timeStr := strings.TrimPrefix(part, "time=")
			if duration := fp.parseTime(timeStr); duration > 0 {
				progress.Time = timeStr
				if totalDuration > 0 {
					progress.Progress = (duration / totalDuration) * 100
				}
			}
		} else if strings.HasPrefix(part, "speed=") {
			speedStr := strings.TrimSuffix(strings.TrimPrefix(part, "speed="), "x")
			if speed, err := strconv.ParseFloat(speedStr, 64); err == nil {
				progress.Speed = speed
			}
		} else if strings.HasPrefix(part, "size=") {
			progress.Bitrate = strings.TrimPrefix(part, "size=")
		}
	}

	return progress
}

func (fp *FFmpegProcessor) parseTime(timeStr string) float64 {
	// Parse time format HH:MM:SS.MS
	parts := strings.Split(timeStr, ":")
	if len(parts) != 3 {
		return 0
	}

	hours, _ := strconv.Atoi(parts[0])
	minutes, _ := strconv.Atoi(parts[1])
	seconds, _ := strconv.ParseFloat(parts[2], 64)

	return float64(hours*3600 + minutes*60) + seconds
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

func (fp *FFmpegProcessor) AddWatermark(inputPath, watermarkPath, outputPath string, position string) error {
	var overlayFilter string
	switch position {
	case "top-left":
		overlayFilter = "overlay=10:10"
	case "top-right":
		overlayFilter = "overlay=main_w-overlay_w-10:10"
	case "bottom-left":
		overlayFilter = "overlay=10:main_h-overlay_h-10"
	case "bottom-right":
		overlayFilter = "overlay=main_w-overlay_w-10:main_h-overlay_h-10"
	case "center":
		overlayFilter = "overlay=(main_w-overlay_w)/2:(main_h-overlay_h)/2"
	default:
		overlayFilter = "overlay=10:10"
	}

	cmd := exec.Command(fp.ffmpegPath,
		"-i", inputPath,
		"-i", watermarkPath,
		"-filter_complex", overlayFilter,
		"-c:a", "copy",
		"-y",
		outputPath,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add watermark: %w", err)
	}

	return nil
}

// Helper type for progress scanning
type ProgressScanner struct {
	*os.File
}

func NewProgressScanner(f *os.File) *ProgressScanner {
	return &ProgressScanner{f}
}

func (ps *ProgressScanner) Scan() bool {
	return true // Simplified implementation
}

func (ps *ProgressScanner) Text() string {
	return "" // Simplified implementation
}