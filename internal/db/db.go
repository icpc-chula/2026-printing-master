package db

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"printingmaster/internal/models"
)

func Connect(dsn string) (*gorm.DB, error) {
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	if err := database.AutoMigrate(&models.Worker{}, &models.Transaction{}); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	return database, nil
}
