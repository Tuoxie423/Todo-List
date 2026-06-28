package model

import "time"

type User struct {
	ID            int       `gorm:"primaryKey" json:"id"`
	Username      string    `gorm:"type:varchar(40);not null;uniqueIndex" json:"username"`
	OpenID        *string   `gorm:"type:varchar(64);uniqueIndex" json:"-"`
	PasswordHash  string    `gorm:"type:varchar(100);not null" json:"-"`
	AuthTokenHash string    `gorm:"type:char(64);index" json:"-"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
