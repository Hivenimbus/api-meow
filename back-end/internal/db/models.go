package db

import "time"

type Instance struct {
	ID              string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name            string    `gorm:"uniqueIndex;not null"`
	Status          string    `gorm:"not null;default:'disconnected';index"`
	PhoneNumber     *string   `gorm:"type:text;index"`
	TagID           *string   `gorm:"type:text"`
	IgnoreGroups    *bool     `gorm:"default:true"`
	WebhookUrl      *string   `gorm:"type:text"`
	ReceiveMessages *bool     `gorm:"default:true"`
	ProxyEnabled    *bool     `gorm:"default:false"`
	ProxyUrl        *string   `gorm:"type:text"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Tag struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string    `gorm:"not null"`
	Color     string    `gorm:"not null"`
	CreatedAt time.Time
}
