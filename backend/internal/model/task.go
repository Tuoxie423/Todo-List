package model

import "time"

type Task struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	RoomID    int       `gorm:"not null;index" json:"roomId"`
	Title     string    `gorm:"type:varchar(100);not null" json:"title"`
	Level     string    `gorm:"type:varchar(20);not null;default:'有空'" json:"level"`
	Done      bool      `gorm:"not null;default:false" json:"done"`
	CreatedAt time.Time `json:"createdAt"`
}
