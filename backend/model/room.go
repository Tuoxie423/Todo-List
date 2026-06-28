package model

import "time"

type Room struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	UserID    int       `gorm:"not null;index;uniqueIndex:idx_user_room_name" json:"userId"`
	Name      string    `gorm:"type:varchar(100);not null;uniqueIndex:idx_user_room_name" json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}
