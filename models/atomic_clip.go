package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type AtomicClip struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Title       string    `json:"title" gorm:"not null;size:200"`
	Description string    `json:"description" gorm:"type:text"`
	FilePath    string    `json:"file_path" gorm:"not null;size:500"`
	FileSize    int64     `json:"file_size"`
	Duration    float64   `json:"duration"`
	Resolution  string    `json:"resolution" gorm:"size:20"`
	FrameRate   float64   `json:"frame_rate"`
	Codec       string    `json:"codec" gorm:"size:50"`
	Bitrate     int       `json:"bitrate"`
	Format      string    `json:"format" gorm:"size:20"`
	Thumbnail   string    `json:"thumbnail" gorm:"size:500"`
	
	// Classification fields
	Category    string    `json:"category" gorm:"size:50"`
	Tags        StringArray `json:"tags" gorm:"type:text"`
	Mood        string    `json:"mood" gorm:"size:50"`
	Style       string    `json:"style" gorm:"size:50"`
	Color       string    `json:"color" gorm:"size:50"`
	
	// AI Analysis fields
	SceneType   string    `json:"scene_type" gorm:"size:50"`
	Objects     StringArray `json:"objects" gorm:"type:text"`
	Actions     StringArray `json:"actions" gorm:"type:text"`
	Emotions    StringArray `json:"emotions" gorm:"type:text"`
	
	// Metadata
	Metadata    JSON      `json:"metadata" gorm:"type:jsonb"`
	
	// Status and relations
	Status      string    `json:"status" gorm:"default:'active';size:20"`
	UserID      uint      `json:"user_id" gorm:"not null"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relations
	User         User           `json:"user,omitempty" gorm:"foreignKey:UserID"`
	VideoAnalysis *VideoAnalysis `json:"video_analysis,omitempty" gorm:"foreignKey:AtomicClipID"`
}

type AtomicClipCreateRequest struct {
	Title       string      `json:"title" binding:"required,max=200"`
	Description string      `json:"description" binding:"omitempty,max=1000"`
	Category    string      `json:"category" binding:"omitempty,max=50"`
	Tags        []string    `json:"tags" binding:"omitempty"`
	Mood        string      `json:"mood" binding:"omitempty,max=50"`
	Style       string      `json:"style" binding:"omitempty,max=50"`
	Color       string      `json:"color" binding:"omitempty,max=50"`
}

type AtomicClipUpdateRequest struct {
	Title       string   `json:"title" binding:"omitempty,max=200"`
	Description string   `json:"description" binding:"omitempty,max=1000"`
	Category    string   `json:"category" binding:"omitempty,max=50"`
	Tags        []string `json:"tags" binding:"omitempty"`
	Mood        string   `json:"mood" binding:"omitempty,max=50"`
	Style       string   `json:"style" binding:"omitempty,max=50"`
	Color       string   `json:"color" binding:"omitempty,max=50"`
}

type AtomicClipSearchRequest struct {
	Query      string   `json:"query" form:"query"`
	Category   string   `json:"category" form:"category"`
	Tags       []string `json:"tags" form:"tags"`
	Mood       string   `json:"mood" form:"mood"`
	Style      string   `json:"style" form:"style"`
	Color      string   `json:"color" form:"color"`
	Duration   string   `json:"duration" form:"duration"` // "short", "medium", "long"
	Resolution string   `json:"resolution" form:"resolution"`
	Page       int      `json:"page" form:"page,default=1"`
	Limit      int      `json:"limit" form:"limit,default=20"`
}

// Custom types for PostgreSQL arrays and JSON
type StringArray []string

func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, s)
	case string:
		return json.Unmarshal([]byte(v), s)
	}
	return nil
}

type JSON map[string]interface{}

func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, j)
	case string:
		return json.Unmarshal([]byte(v), j)
	}
	return nil
}