package model

import "time"

type Room struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"type:varchar(100);not null;uniqueIndex" json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}
