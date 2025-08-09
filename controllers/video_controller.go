package controllers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"creative-studio-server/config"
	"creative-studio-server/pkg/logger"
	"creative-studio-server/pkg/video_engine"
)

type VideoController struct {
	ffmpegProcessor *video_engine.FFmpegProcessor
}

func NewVideoController() *VideoController {
	cfg := config.AppConfig
	return &VideoController{
		ffmpegProcessor: video_engine.NewFFmpegProcessor(cfg),
	}
}

// 上传视频文件
func (vc *VideoController) UploadVideo(c *gin.Context) {
	// 解析表单数据
	err := c.Request.ParseMultipartForm(500 << 20) // 500MB max
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to parse form data",
		})
		return
	}

	// 获取上传的文件
	file, header, err := c.Request.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No video file provided",
		})
		return
	}
	defer file.Close()

	// 验证文件类型
	contentType := header.Header.Get("Content-Type")
	if !isValidVideoType(contentType) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid file type. Only video files are allowed",
		})
		return
	}

	// 创建上传目录
	uploadDir := "./uploads"
	os.MkdirAll(uploadDir, 0755)

	// 生成文件名
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("%d_%s", timestamp, header.Filename)
	filePath := filepath.Join(uploadDir, filename)

	// 保存文件
	dst, err := os.Create(filePath)
	if err != nil {
		logger.Errorf("Failed to create file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save file",
		})
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		logger.Errorf("Failed to save file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save file",
		})
		return
	}

	// 获取视频信息
	videoInfo, err := vc.ffmpegProcessor.GetVideoInfo(filePath)
	if err != nil {
		logger.Errorf("Failed to get video info: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to analyze video",
		})
		return
	}

	logger.Infof("Video uploaded successfully: %s", filename)

	c.JSON(http.StatusOK, gin.H{
		"message":    "Video uploaded successfully",
		"filename":   filename,
		"filepath":   filePath,
		"video_info": videoInfo,
	})
}

// 拼接视频
func (vc *VideoController) ConcatenateVideos(c *gin.Context) {
	var request struct {
		Files []string `json:"files" binding:"required"`
		OutputName string `json:"output_name"`
		Quality string `json:"quality"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
		})
		return
	}

	if len(request.Files) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "At least 2 files are required for concatenation",
		})
		return
	}

	// 验证文件存在
	var inputPaths []string
	for _, filename := range request.Files {
		filePath := filepath.Join("./uploads", filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("File not found: %s", filename),
			})
			return
		}
		inputPaths = append(inputPaths, filePath)
	}

	// 生成输出文件名
	outputName := request.OutputName
	if outputName == "" {
		outputName = fmt.Sprintf("concat_%d.mp4", time.Now().Unix())
	}
	if filepath.Ext(outputName) == "" {
		outputName += ".mp4"
	}

	outputPath := filepath.Join("./output", outputName)
	os.MkdirAll("./output", 0755)

	// 设置渲染选项
	options := &video_engine.RenderOptions{
		OutputFormat: "mp4",
		Quality:      getQualityOrDefault(request.Quality),
		Preset:       "medium",
	}

	logger.Infof("Starting video concatenation: %v -> %s", request.Files, outputName)

	// 执行拼接
	err := vc.ffmpegProcessor.ConcatenateVideos(inputPaths, outputPath, options)
	if err != nil {
		logger.Errorf("Failed to concatenate videos: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to concatenate videos",
			"details": err.Error(),
		})
		return
	}

	// 获取输出文件信息
	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		logger.Errorf("Failed to get output file info: %v", err)
	}

	logger.Infof("Video concatenation completed: %s", outputName)

	c.JSON(http.StatusOK, gin.H{
		"message":     "Videos concatenated successfully",
		"output_file": outputName,
		"output_path": outputPath,
		"file_size":   fileInfo.Size(),
		"download_url": fmt.Sprintf("/api/v1/video/download/%s", outputName),
	})
}

// 下载拼接后的视频
func (vc *VideoController) DownloadVideo(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Filename is required",
		})
		return
	}

	filePath := filepath.Join("./output", filename)
	
	// 验证文件存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "File not found",
		})
		return
	}

	// 设置响应头
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/octet-stream")

	// 发送文件
	c.File(filePath)
}

// 列出已上传的文件
func (vc *VideoController) ListFiles(c *gin.Context) {
	uploadDir := "./uploads"
	files, err := os.ReadDir(uploadDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read upload directory",
		})
		return
	}

	var videoFiles []map[string]interface{}
	for _, file := range files {
		if !file.IsDir() {
			filePath := filepath.Join(uploadDir, file.Name())
			info, _ := file.Info()
			
			videoFiles = append(videoFiles, map[string]interface{}{
				"name":      file.Name(),
				"size":      info.Size(),
				"modified":  info.ModTime(),
				"path":      filePath,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"files": videoFiles,
		"count": len(videoFiles),
	})
}

// 列出已生成的输出文件
func (vc *VideoController) ListOutputFiles(c *gin.Context) {
	outputDir := "./output"
	files, err := os.ReadDir(outputDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read output directory",
		})
		return
	}

	var outputFiles []map[string]interface{}
	for _, file := range files {
		if !file.IsDir() {
			info, _ := file.Info()
			
			outputFiles = append(outputFiles, map[string]interface{}{
				"name":         file.Name(),
				"size":         info.Size(),
				"modified":     info.ModTime(),
				"download_url": fmt.Sprintf("/api/v1/video/download/%s", file.Name()),
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"files": outputFiles,
		"count": len(outputFiles),
	})
}

// 删除文件
func (vc *VideoController) DeleteFile(c *gin.Context) {
	filename := c.Param("filename")
	fileType := c.Query("type") // "upload" or "output"
	
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Filename is required",
		})
		return
	}

	var filePath string
	if fileType == "output" {
		filePath = filepath.Join("./output", filename)
	} else {
		filePath = filepath.Join("./uploads", filename)
	}

	// 删除文件
	err := os.Remove(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "File not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to delete file",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "File deleted successfully",
	})
}

// 获取视频信息
func (vc *VideoController) GetVideoInfo(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Filename is required",
		})
		return
	}

	filePath := filepath.Join("./uploads", filename)
	
	// 验证文件存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "File not found",
		})
		return
	}

	// 获取视频信息
	videoInfo, err := vc.ffmpegProcessor.GetVideoInfo(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to analyze video",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"filename":    filename,
		"video_info": videoInfo,
	})
}

// 健康检查
func (vc *VideoController) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"service": "video-processor",
		"timestamp": time.Now(),
		"ffmpeg_available": true,
	})
}

// 辅助函数
func isValidVideoType(contentType string) bool {
	validTypes := []string{
		"video/mp4",
		"video/quicktime",
		"video/x-msvideo",
		"video/x-matroska",
		"video/webm",
	}

	for _, validType := range validTypes {
		if contentType == validType {
			return true
		}
	}
	return false
}

func getQualityOrDefault(quality string) string {
	validQualities := []string{"low", "medium", "high", "ultra"}
	for _, valid := range validQualities {
		if quality == valid {
			return quality
		}
	}
	return "medium" // default
}