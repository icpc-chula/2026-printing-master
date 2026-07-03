package models

import (
	"time"
)

type Transaction struct {
	ID        uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	Username  string     `json:"username" gorm:"not null"`
	TeamName  string     `json:"team_name" gorm:"not null"`
	TeamID    string     `json:"team_id" gorm:"not null"`
	Location  string     `json:"location" gorm:"not null"`
	FileName  string     `json:"file_name" gorm:"not null"`
	FilePath  string     `json:"file_path" gorm:"not null"`
	Pages     int        `json:"pages" gorm:"not null"`
	WorkerID  uint       `json:"worker_id" gorm:"not null"`
	Worker    Worker     `json:"worker" gorm:"foreignKey:WorkerID"`
	Status    string     `json:"status" gorm:"not null;default:'pending'"`
	PrintedAt *time.Time `json:"printed_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type Worker struct {
	ID        uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	IPAddress string `json:"ip_address" gorm:"not null;uniqueIndex"`
}

const (
	StatusPending = "pending"
	StatusPrinted = "printed"
	StatusFailed  = "failed"
)
