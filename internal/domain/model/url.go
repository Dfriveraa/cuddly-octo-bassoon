package model

import (
"time"
)

// URL representa la entidad principal de nuestro dominio para el acortador de URLs
type URL struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	OriginalURL string   `json:"original_url" gorm:"type:text;not null"`
	ShortCode  string   `json:"short_code" gorm:"type:varchar(10);uniqueIndex;not null"`
	Visits     int      `json:"visits" gorm:"default:0"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}
