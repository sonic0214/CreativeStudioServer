package routes

import (
	"github.com/gin-gonic/gin"
	"creative-studio-server/controllers"
)

func SetupRoutes(r *gin.Engine) {
	// Initialize video controller
	videoController := controllers.NewVideoController()

	// Health check and system endpoints
	r.GET("/health", healthCheck)
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Creative Studio Video Server API",
			"version": "1.0.0",
			"status":  "running",
		})
	})

	// API v1 routes - simplified for video processing only
	v1 := r.Group("/api/v1")
	{
		// Video processing routes (no authentication required)
		videos := v1.Group("/videos")
		{
			videos.POST("/upload", videoController.UploadVideo)
			videos.POST("/concatenate", videoController.ConcatenateVideos)
			videos.GET("/files", videoController.ListFiles)
			videos.GET("/output", videoController.ListOutputFiles)
			videos.GET("/info/:filename", videoController.GetVideoInfo)
			videos.GET("/download/:filename", videoController.DownloadVideo)
			videos.DELETE("/:filename", videoController.DeleteFile)
		}
	}
}

func healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":    "healthy",
		"timestamp": gin.H{
			"unix":      gin.H{"seconds": 1234567890},
			"formatted": "2023-12-07T10:00:00Z",
		},
		"services": gin.H{
			"ffmpeg": "available",
		},
	})
}