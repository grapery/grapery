package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/grapestree/fgrapery/grapery/internal/config"
	"github.com/grapestree/fgrapery/grapery/internal/repository/migrations"
	_ "github.com/grapestree/fgrapery/grapery/internal/repository/mysql" // Register migrations
	_ "github.com/grapestree/fgrapery/grapery/internal/repository/pay"   // Register payment migrations
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// Load configuration from environment variables
	cfg := config.Load("migration-runner")

	// Build DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=True&loc=Local",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Address,
		cfg.Database.Database,
	)

	// Connect to database
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				LogLevel: logger.Info,
			},
		),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Get SQL DB instance to test connection
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get SQL DB instance: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("Successfully connected to database!")

	// Create logger
	zapLogger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer zapLogger.Sync()

	// Get registry and run all migrations
	registry := migrations.GetRegistry()

	ctx := context.Background()
	if err := registry.ExecuteAll(ctx, db, zapLogger); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	fmt.Println("\n✅ All migrations completed successfully!")
}
