package models

import (
	"time"

	"gorm.io/gorm"
)

type RenderTask struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	TaskID       string    `json:"task_id" gorm:"uniqueIndex;not null;size:50"`
	
	// Task details
	Status       string    `json:"status" gorm:"default:'pending';size:20"`
	Progress     int       `json:"progress" gorm:"default:0"`
	Priority     int       `json:"priority" gorm:"default:5"`
	
	// Render settings
	OutputFormat string    `json:"output_format" gorm:"size:20"`
	Quality      string    `json:"quality" gorm:"size:20"`
	Resolution   string    `json:"resolution" gorm:"size:20"`
	FrameRate    float64   `json:"frame_rate"`
	
	// File information
	OutputPath   string    `json:"output_path" gorm:"size:500"`
	FileSize     int64     `json:"file_size"`
	Duration     float64   `json:"duration"`
	
	// Timing information
	StartedAt    *time.Time `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at"`
	EstimatedTime int       `json:"estimated_time"` // in seconds
	
	// Error information
	ErrorMessage string    `json:"error_message" gorm:"type:text"`
	RetryCount   int       `json:"retry_count" gorm:"default:0"`
	
	// Relations
	ProjectID    uint      `json:"project_id" gorm:"not null"`
	UserID       uint      `json:"user_id" gorm:"not null"`
	
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relations
	Project      Project   `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	User         User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

type RenderTaskCreateRequest struct {
	ProjectID    uint    `json:"project_id" binding:"required"`
	OutputFormat string  `json:"output_format" binding:"required,oneof=mp4 mov avi mkv"`
	Quality      string  `json:"quality" binding:"required,oneof=low medium high ultra"`
	Resolution   string  `json:"resolution" binding:"omitempty"`
	FrameRate    float64 `json:"frame_rate" binding:"omitempty,min=1,max=120"`
	Priority     int     `json:"priority" binding:"omitempty,min=1,max=10"`
}

type RenderTaskUpdateRequest struct {
	Status       string  `json:"status" binding:"omitempty,oneof=pending processing completed failed cancelled"`
	Progress     int     `json:"progress" binding:"omitempty,min=0,max=100"`
	Priority     int     `json:"priority" binding:"omitempty,min=1,max=10"`
	ErrorMessage string  `json:"error_message" binding:"omitempty"`
}

type VideoAnalysis struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	AtomicClipID  uint      `json:"atomic_clip_id" gorm:"not null;uniqueIndex"`
	
	// Technical analysis
	AvgBrightness float64   `json:"avg_brightness"`
	AvgContrast   float64   `json:"avg_contrast"`
	AvgSaturation float64   `json:"avg_saturation"`
	DominantColors StringArray `json:"dominant_colors" gorm:"type:text"`
	
	// Motion analysis
	MotionIntensity string  `json:"motion_intensity" gorm:"size:20"` // low, medium, high
	CameraMovement  string  `json:"camera_movement" gorm:"size:50"`
	
	// Content analysis
	HasFaces      bool      `json:"has_faces"`
	FaceCount     int       `json:"face_count"`
	HasText       bool      `json:"has_text"`
	TextContent   string    `json:"text_content" gorm:"type:text"`
	
	// Audio analysis (if available)
	HasAudio      bool      `json:"has_audio"`
	AudioLevel    float64   `json:"audio_level"`
	AudioType     string    `json:"audio_type" gorm:"size:50"` // music, speech, sfx, silence
	
	// AI-generated data
	AITags        StringArray `json:"ai_tags" gorm:"type:text"`
	AIDescription string    `json:"ai_description" gorm:"type:text"`
	Confidence    float64   `json:"confidence"`
	
	// Analysis metadata
	AnalysisVersion string  `json:"analysis_version" gorm:"size:20"`
	ProcessedAt   time.Time `json:"processed_at"`
	
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	
	// Relations
	AtomicClip    AtomicClip `json:"atomic_clip,omitempty" gorm:"foreignKey:AtomicClipID"`
}