package model

import "time"

// User represents a user in the system
type User struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"unique"`
	Email     string    `json:"email" gorm:"unique"`
	Password  string    `json:"-"` // The "-" tag excludes this field from JSON responses
	CreatedAt time.Time `json:"created_at"`
	Devices   []Device  `json:"devices,omitempty" gorm:"foreignKey:UserID"` // A user can have multiple devices
}
