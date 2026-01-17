package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"balanca/internal/config"
	md "balanca/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect(cfg *config.DatabaseConfig) error {
	// dsn := fmt.Sprintf(
	// 	"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
	// 	cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode,
	// )
dsn := cfg.DBURL
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
			Colorful:                  true,
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	DB = db
	return nil
}

func AutoMigrate() error {
	models := []interface{}{
		&md.User{},
		&md.Group{},
		&md.UserGroup{},
		&md.Transaction{},
		&md.PlannedExpense{},
		&md.AuditLog{},
		&md.Notification{},
	}

	if err := DB.AutoMigrate(models...); err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	return nil
}

func GetDB() *gorm.DB {
	return DB
}
