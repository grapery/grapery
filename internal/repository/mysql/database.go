package mysql

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const (
	maxIdleConns    = 10
	maxOpenConns    = 100
	connMaxLifetime = 3600
)

// InitDB creates a new GORM database connection from a DSN string
func InitDB(dsn string, log *zap.Logger) (*gorm.DB, error) {
	newLogger := gormlogger.New(
		&zapAdapter{log: log},
		gormlogger.Config{
			SlowThreshold:             time.Second * 5,
			LogLevel:                  gormlogger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("connect database failed: %w", err)
	}

	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Second)

	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn: sqlDB,
	}), &gorm.Config{Logger: newLogger})
	if err != nil {
		return nil, fmt.Errorf("create orm failed: %w", err)
	}

	return db, nil
}

// zapAdapter adapts zap.Logger to gorm logger interface
type zapAdapter struct {
	log *zap.Logger
}

func (z *zapAdapter) Printf(format string, args ...interface{}) {
	z.log.Sugar().Infof(format, args...)
}
