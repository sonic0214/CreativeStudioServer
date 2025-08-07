package services

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"creative-studio-server/models"
	"creative-studio-server/pkg/database"
	"creative-studio-server/pkg/logger"
)

type AtomicClipService struct {
	db *gorm.DB
}

func NewAtomicClipService() *AtomicClipService {
	return &AtomicClipService{
		db: database.GetDB(),
	}
}

func (s *AtomicClipService) CreateAtomicClip(userID uint, req *models.AtomicClipCreateRequest, filePath string, fileInfo map[string]interface{}) (*models.AtomicClip, error) {
	clip := &models.AtomicClip{
		Title:       req.Title,
		Description: req.Description,
		FilePath:    filePath,
		Category:    req.Category,
		Tags:        req.Tags,
		Mood:        req.Mood,
		Style:       req.Style,
		Color:       req.Color,
		UserID:      userID,
		Status:      "active",
	}

	// Set file information from analysis
	if size, ok := fileInfo["file_size"].(int64); ok {
		clip.FileSize = size
	}
	if duration, ok := fileInfo["duration"].(float64); ok {
		clip.Duration = duration
	}
	if resolution, ok := fileInfo["resolution"].(string); ok {
		clip.Resolution = resolution
	}
	if frameRate, ok := fileInfo["frame_rate"].(float64); ok {
		clip.FrameRate = frameRate
	}
	if codec, ok := fileInfo["codec"].(string); ok {
		clip.Codec = codec
	}
	if bitrate, ok := fileInfo["bitrate"].(int); ok {
		clip.Bitrate = bitrate
	}
	if format, ok := fileInfo["format"].(string); ok {
		clip.Format = format
	}
	if thumbnail, ok := fileInfo["thumbnail"].(string); ok {
		clip.Thumbnail = thumbnail
	}

	if err := s.db.Create(clip).Error; err != nil {
		logger.Errorf("Failed to create atomic clip: %v", err)
		return nil, errors.New("failed to create atomic clip")
	}

	logger.Infof("Atomic clip created successfully: %d", clip.ID)
	return clip, nil
}

func (s *AtomicClipService) GetAtomicClipByID(clipID, userID uint) (*models.AtomicClip, error) {
	var clip models.AtomicClip
	query := s.db.Preload("User").Preload("VideoAnalysis")
	
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	
	if err := query.First(&clip, clipID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("atomic clip not found")
		}
		logger.Errorf("Failed to get atomic clip: %v", err)
		return nil, errors.New("failed to get atomic clip")
	}

	return &clip, nil
}

func (s *AtomicClipService) UpdateAtomicClip(clipID, userID uint, req *models.AtomicClipUpdateRequest) (*models.AtomicClip, error) {
	var clip models.AtomicClip
	if err := s.db.Where("id = ? AND user_id = ?", clipID, userID).First(&clip).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("atomic clip not found")
		}
		return nil, errors.New("failed to get atomic clip")
	}

	// Update fields
	if req.Title != "" {
		clip.Title = req.Title
	}
	if req.Description != "" {
		clip.Description = req.Description
	}
	if req.Category != "" {
		clip.Category = req.Category
	}
	if len(req.Tags) > 0 {
		clip.Tags = req.Tags
	}
	if req.Mood != "" {
		clip.Mood = req.Mood
	}
	if req.Style != "" {
		clip.Style = req.Style
	}
	if req.Color != "" {
		clip.Color = req.Color
	}

	if err := s.db.Save(&clip).Error; err != nil {
		logger.Errorf("Failed to update atomic clip: %v", err)
		return nil, errors.New("failed to update atomic clip")
	}

	return &clip, nil
}

func (s *AtomicClipService) DeleteAtomicClip(clipID, userID uint) error {
	result := s.db.Where("id = ? AND user_id = ?", clipID, userID).Delete(&models.AtomicClip{})
	if result.Error != nil {
		logger.Errorf("Failed to delete atomic clip: %v", result.Error)
		return errors.New("failed to delete atomic clip")
	}
	
	if result.RowsAffected == 0 {
		return errors.New("atomic clip not found")
	}

	return nil
}

func (s *AtomicClipService) SearchAtomicClips(req *models.AtomicClipSearchRequest, userID uint) ([]models.AtomicClip, int64, error) {
	var clips []models.AtomicClip
	var total int64

	query := s.db.Model(&models.AtomicClip{}).Preload("User").Preload("VideoAnalysis")
	
	// Filter by user if specified
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	// Apply search filters
	if req.Query != "" {
		searchTerm := "%" + strings.ToLower(req.Query) + "%"
		query = query.Where("LOWER(title) LIKE ? OR LOWER(description) LIKE ?", searchTerm, searchTerm)
	}

	if req.Category != "" {
		query = query.Where("category = ?", req.Category)
	}

	if req.Mood != "" {
		query = query.Where("mood = ?", req.Mood)
	}

	if req.Style != "" {
		query = query.Where("style = ?", req.Style)
	}

	if req.Color != "" {
		query = query.Where("color = ?", req.Color)
	}

	if req.Resolution != "" {
		query = query.Where("resolution = ?", req.Resolution)
	}

	if len(req.Tags) > 0 {
		for _, tag := range req.Tags {
			query = query.Where("tags::text ILIKE ?", "%"+tag+"%")
		}
	}

	// Duration filter
	switch req.Duration {
	case "short":
		query = query.Where("duration < ?", 30) // Less than 30 seconds
	case "medium":
		query = query.Where("duration >= ? AND duration <= ?", 30, 180) // 30 seconds to 3 minutes
	case "long":
		query = query.Where("duration > ?", 180) // More than 3 minutes
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count atomic clips: %w", err)
	}

	// Apply pagination
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100 // Max limit
	}

	offset := (req.Page - 1) * req.Limit
	if err := query.Offset(offset).Limit(req.Limit).Order("created_at DESC").Find(&clips).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get atomic clips: %w", err)
	}

	return clips, total, nil
}

func (s *AtomicClipService) GetUserAtomicClips(userID uint, page, limit int) ([]models.AtomicClip, int64, error) {
	var clips []models.AtomicClip
	var total int64

	query := s.db.Model(&models.AtomicClip{}).Where("user_id = ?", userID).Preload("VideoAnalysis")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count atomic clips: %w", err)
	}

	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&clips).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get atomic clips: %w", err)
	}

	return clips, total, nil
}

func (s *AtomicClipService) GetSimilarClips(clipID uint, limit int) ([]models.AtomicClip, error) {
	var baseClip models.AtomicClip
	if err := s.db.First(&baseClip, clipID).Error; err != nil {
		return nil, errors.New("clip not found")
	}

	var clips []models.AtomicClip
	query := s.db.Model(&models.AtomicClip{}).
		Where("id != ?", clipID).
		Preload("VideoAnalysis")

	// Find similar clips based on category, mood, style, or tags
	if baseClip.Category != "" {
		query = query.Where("category = ?", baseClip.Category)
	}
	if baseClip.Mood != "" {
		query = query.Where("mood = ?", baseClip.Mood)
	}
	if baseClip.Style != "" {
		query = query.Where("style = ?", baseClip.Style)
	}

	if err := query.Limit(limit).Order("created_at DESC").Find(&clips).Error; err != nil {
		return nil, fmt.Errorf("failed to get similar clips: %w", err)
	}

	return clips, nil
}