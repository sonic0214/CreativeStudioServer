package controllers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"creative-studio-server/middleware"
	"creative-studio-server/models"
	"creative-studio-server/services"
	"creative-studio-server/pkg/logger"
)

type AtomicClipController struct {
	atomicClipService *services.AtomicClipService
}

func NewAtomicClipController() *AtomicClipController {
	return &AtomicClipController{
		atomicClipService: services.NewAtomicClipService(),
	}
}

// @Summary Create atomic clip
// @Description Upload and create a new atomic clip
// @Tags atomic-clips
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param title formData string true "Clip title"
// @Param description formData string false "Clip description"
// @Param category formData string false "Clip category"
// @Param tags formData string false "Clip tags (comma-separated)"
// @Param mood formData string false "Clip mood"
// @Param style formData string false "Clip style"
// @Param color formData string false "Clip color"
// @Param video formData file true "Video file"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/atomic-clips [post]
func (c *AtomicClipController) CreateAtomicClip(ctx *gin.Context) {
	userID, exists := middleware.GetUserID(ctx)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// Parse multipart form
	err := ctx.Request.ParseMultipartForm(100 << 20) // 100MB max
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to parse multipart form",
		})
		return
	}

	// Get file
	file, header, err := ctx.Request.FormFile("video")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Video file is required",
		})
		return
	}
	defer file.Close()

	// Validate file type
	contentType := header.Header.Get("Content-Type")
	if contentType != "video/mp4" && contentType != "video/quicktime" && 
	   contentType != "video/x-msvideo" && contentType != "video/x-matroska" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid file type. Only video files are allowed",
		})
		return
	}

	// Create request from form data
	req := &models.AtomicClipCreateRequest{
		Title:       ctx.Request.FormValue("title"),
		Description: ctx.Request.FormValue("description"),
		Category:    ctx.Request.FormValue("category"),
		Mood:        ctx.Request.FormValue("mood"),
		Style:       ctx.Request.FormValue("style"),
		Color:       ctx.Request.FormValue("color"),
	}

	// Parse tags if provided
	if tagsStr := ctx.Request.FormValue("tags"); tagsStr != "" {
		// In a real implementation, you'd parse comma-separated tags
		req.Tags = []string{tagsStr}
	}

	// Validate request
	if req.Title == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Title is required",
		})
		return
	}

	// TODO: Process file upload, save to storage, and analyze video
	// For now, we'll create a placeholder implementation
	filePath := fmt.Sprintf("/uploads/clips/%d_%s", userID, header.Filename)
	fileInfo := map[string]interface{}{
		"file_size":  header.Size,
		"duration":   60.0, // Placeholder
		"resolution": "1920x1080", // Placeholder
		"frame_rate": 30.0, // Placeholder
		"codec":      "h264", // Placeholder
		"bitrate":    2000, // Placeholder
		"format":     "mp4", // Placeholder
	}

	clip, err := c.atomicClipService.CreateAtomicClip(userID, req, filePath, fileInfo)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "Atomic clip created successfully",
		"clip":    clip,
	})
}

// @Summary Get atomic clip by ID
// @Description Retrieve a specific atomic clip by ID
// @Tags atomic-clips
// @Produce json
// @Security BearerAuth
// @Param id path int true "Clip ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/atomic-clips/{id} [get]
func (c *AtomicClipController) GetAtomicClip(ctx *gin.Context) {
	clipID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid clip ID",
		})
		return
	}

	userID, _ := middleware.GetUserID(ctx)
	
	clip, err := c.atomicClipService.GetAtomicClipByID(uint(clipID), userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "atomic clip not found" {
			statusCode = http.StatusNotFound
		}
		ctx.JSON(statusCode, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"clip": clip,
	})
}

// @Summary Update atomic clip
// @Description Update an existing atomic clip
// @Tags atomic-clips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Clip ID"
// @Param clip body models.AtomicClipUpdateRequest true "Updated clip data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/atomic-clips/{id} [put]
func (c *AtomicClipController) UpdateAtomicClip(ctx *gin.Context) {
	clipID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid clip ID",
		})
		return
	}

	userID, exists := middleware.GetUserID(ctx)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req models.AtomicClipUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	clip, err := c.atomicClipService.UpdateAtomicClip(uint(clipID), userID, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "atomic clip not found" {
			statusCode = http.StatusNotFound
		}
		ctx.JSON(statusCode, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Atomic clip updated successfully",
		"clip":    clip,
	})
}

// @Summary Delete atomic clip
// @Description Delete an atomic clip
// @Tags atomic-clips
// @Produce json
// @Security BearerAuth
// @Param id path int true "Clip ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/atomic-clips/{id} [delete]
func (c *AtomicClipController) DeleteAtomicClip(ctx *gin.Context) {
	clipID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid clip ID",
		})
		return
	}

	userID, exists := middleware.GetUserID(ctx)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	err = c.atomicClipService.DeleteAtomicClip(uint(clipID), userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "atomic clip not found" {
			statusCode = http.StatusNotFound
		}
		ctx.JSON(statusCode, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Atomic clip deleted successfully",
	})
}

// @Summary Search atomic clips
// @Description Search and filter atomic clips
// @Tags atomic-clips
// @Produce json
// @Security BearerAuth
// @Param query query string false "Search query"
// @Param category query string false "Filter by category"
// @Param mood query string false "Filter by mood"
// @Param style query string false "Filter by style"
// @Param color query string false "Filter by color"
// @Param duration query string false "Filter by duration (short/medium/long)"
// @Param resolution query string false "Filter by resolution"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/atomic-clips/search [get]
func (c *AtomicClipController) SearchAtomicClips(ctx *gin.Context) {
	var req models.AtomicClipSearchRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid query parameters",
			"details": err.Error(),
		})
		return
	}

	userID, _ := middleware.GetUserID(ctx)
	
	clips, total, err := c.atomicClipService.SearchAtomicClips(&req, userID)
	if err != nil {
		logger.Errorf("Failed to search atomic clips: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to search atomic clips",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"clips": clips,
		"pagination": gin.H{
			"page":  req.Page,
			"limit": req.Limit,
			"total": total,
			"pages": (total + int64(req.Limit) - 1) / int64(req.Limit),
		},
	})
}

// @Summary Get user's atomic clips
// @Description Get all atomic clips for the authenticated user
// @Tags atomic-clips
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/atomic-clips/my-clips [get]
func (c *AtomicClipController) GetUserAtomicClips(ctx *gin.Context) {
	userID, exists := middleware.GetUserID(ctx)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))

	clips, total, err := c.atomicClipService.GetUserAtomicClips(userID, page, limit)
	if err != nil {
		logger.Errorf("Failed to get user atomic clips: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get atomic clips",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"clips": clips,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// @Summary Get similar clips
// @Description Get clips similar to the specified clip
// @Tags atomic-clips
// @Produce json
// @Security BearerAuth
// @Param id path int true "Clip ID"
// @Param limit query int false "Number of similar clips to return" default(10)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/atomic-clips/{id}/similar [get]
func (c *AtomicClipController) GetSimilarClips(ctx *gin.Context) {
	clipID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid clip ID",
		})
		return
	}

	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	if limit > 50 {
		limit = 50 // Max limit
	}

	clips, err := c.atomicClipService.GetSimilarClips(uint(clipID), limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"clips": clips,
	})
}