package database

import (
	"fmt"

	"electronic-digital-signature/internal/app/config"
	"electronic-digital-signature/internal/domain/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewPostgres(cfg config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.PostgresDSN()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("postgres db is not configured")
	}

	if err := db.AutoMigrate(&model.Message{}, &model.Document{}, &model.User{}); err != nil {
		return fmt.Errorf("auto migrate postgres: %w", err)
	}

	return nil
}
