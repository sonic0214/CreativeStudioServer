package database

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"creative-studio-server/config"
	"creative-studio-server/models"
	pkgLogger "creative-studio-server/pkg/logger"
)

var DB *gorm.DB

func InitDatabase(cfg *config.Config) error {
	var err error

	// Configure GORM logger
	gormLogger := logger.New(
		pkgLogger.Logger,
		logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Info,
			Colorful:      false,
		},
	)

	DB, err = gorm.Open(mysql.Open(cfg.GetDSN()), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifeTime)
	sqlDB.SetConnMaxIdleTime(cfg.Database.MaxIdleTime)

	// Auto-migrate models (disabled - manual table creation)
	// if err := AutoMigrate(); err != nil {
	// 	return fmt.Errorf("failed to auto-migrate models: %w", err)
	// }

	pkgLogger.Info("Database connected successfully")
	return nil
}

func AutoMigrate() error {
	return DB.AutoMigrate(
		&models.User{},
		&models.AtomicClip{},
		&models.Project{},
		&models.Template{},
		&models.RenderTask{},
		&models.VideoAnalysis{},
	)
}

func GetDB() *gorm.DB {
	return DB
}