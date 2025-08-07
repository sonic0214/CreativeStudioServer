package models

import (
	"time"

	"gorm.io/gorm"
)

type Project struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Title       string    `json:"title" gorm:"not null;size:200"`
	Description string    `json:"description" gorm:"type:text"`
	
	// Project settings
	Width       int       `json:"width" gorm:"default:1920"`
	Height      int       `json:"height" gorm:"default:1080"`
	FrameRate   float64   `json:"frame_rate" gorm:"default:30"`
	Duration    float64   `json:"duration"`
	
	// Timeline data (stored as JSON)
	Timeline    JSON      `json:"timeline" gorm:"type:jsonb"`
	Settings    JSON      `json:"settings" gorm:"type:jsonb"`
	
	// Status and metadata
	Status      string    `json:"status" gorm:"default:'draft';size:20"`
	Version     int       `json:"version" gorm:"default:1"`
	Thumbnail   string    `json:"thumbnail" gorm:"size:500"`
	
	// Relations
	UserID      uint      `json:"user_id" gorm:"not null"`
	TemplateID  *uint     `json:"template_id"`
	
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relations
	User        User        `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Template    *Template   `json:"template,omitempty" gorm:"foreignKey:TemplateID"`
	RenderTasks []RenderTask `json:"render_tasks,omitempty" gorm:"foreignKey:ProjectID"`
}

type ProjectCreateRequest struct {
	Title       string  `json:"title" binding:"required,max=200"`
	Description string  `json:"description" binding:"omitempty,max=1000"`
	Width       int     `json:"width" binding:"omitempty,min=320,max=7680"`
	Height      int     `json:"height" binding:"omitempty,min=240,max=4320"`
	FrameRate   float64 `json:"frame_rate" binding:"omitempty,min=1,max=120"`
	TemplateID  *uint   `json:"template_id" binding:"omitempty"`
}

type ProjectUpdateRequest struct {
	Title       string  `json:"title" binding:"omitempty,max=200"`
	Description string  `json:"description" binding:"omitempty,max=1000"`
	Width       int     `json:"width" binding:"omitempty,min=320,max=7680"`
	Height      int     `json:"height" binding:"omitempty,min=240,max=4320"`
	FrameRate   float64 `json:"frame_rate" binding:"omitempty,min=1,max=120"`
	Timeline    JSON    `json:"timeline" binding:"omitempty"`
	Settings    JSON    `json:"settings" binding:"omitempty"`
	Status      string  `json:"status" binding:"omitempty,oneof=draft active archived"`
}

type Template struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"not null;size:200"`
	Description string    `json:"description" gorm:"type:text"`
	Category    string    `json:"category" gorm:"size:50"`
	
	// Template settings
	Width       int       `json:"width" gorm:"default:1920"`
	Height      int       `json:"height" gorm:"default:1080"`
	FrameRate   float64   `json:"frame_rate" gorm:"default:30"`
	Duration    float64   `json:"duration"`
	
	// Template data
	Timeline    JSON      `json:"timeline" gorm:"type:jsonb"`
	Settings    JSON      `json:"settings" gorm:"type:jsonb"`
	Thumbnail   string    `json:"thumbnail" gorm:"size:500"`
	
	// Template metadata
	Tags        StringArray `json:"tags" gorm:"type:text"`
	IsPublic    bool      `json:"is_public" gorm:"default:false"`
	UsageCount  int       `json:"usage_count" gorm:"default:0"`
	
	UserID      uint      `json:"user_id" gorm:"not null"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relations
	User        User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Projects    []Project `json:"projects,omitempty" gorm:"foreignKey:TemplateID"`
}

type TemplateCreateRequest struct {
	Name        string   `json:"name" binding:"required,max=200"`
	Description string   `json:"description" binding:"omitempty,max=1000"`
	Category    string   `json:"category" binding:"omitempty,max=50"`
	Width       int      `json:"width" binding:"omitempty,min=320,max=7680"`
	Height      int      `json:"height" binding:"omitempty,min=240,max=4320"`
	FrameRate   float64  `json:"frame_rate" binding:"omitempty,min=1,max=120"`
	Timeline    JSON     `json:"timeline" binding:"required"`
	Settings    JSON     `json:"settings" binding:"omitempty"`
	Tags        []string `json:"tags" binding:"omitempty"`
	IsPublic    bool     `json:"is_public" binding:"omitempty"`
}