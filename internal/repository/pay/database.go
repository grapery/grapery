package pay

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	logger "gorm.io/gorm/logger"
)

var (
	database       *gorm.DB
	sqlDB          *sql.DB
	logFieldModels = zap.Fields(
		zap.String("module", "models"))
)

const (
	maxIdleConns    = 10
	maxOpenConns    = 20
	connMaxLifetime = 3600
)

// Init ...
func Init(uname, pwd, address, dbname string) error {
	var err error
	if database != nil {
		log.Warn("database already init")
		return nil
	}
	newLogger := logger.New(
		log.StandardLogger(),
		logger.Config{
			SlowThreshold: time.Second * 5,
			LogLevel:      logger.Info,
			Colorful:      true,
		},
	)
	sqldbUrl := fmt.Sprintf("%s:%s@(%s:3306)/%s?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=True&loc=Local", uname, pwd, address, dbname)
	log.Infof("sqldbUrl: %s", sqldbUrl)
	sqlDB, err := sql.Open("mysql", sqldbUrl)

	if err != nil {
		log.Errorf("connect database failed  : [%s]", err.Error())
		return err
	}
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	database, err = gorm.Open(mysql.New(mysql.Config{
		Conn: sqlDB,
	}), &gorm.Config{Logger: newLogger})
	if err != nil {
		log.Errorf("create orm failed  : [%s]", err.Error())
		return err
	}
	database.Callback().Update().Before("gorm:update").Register("update_update_at", callbacks.BeforeCreate)
	// Note: GORM automatically handles soft delete with DeletedAt field, no need for manual deleteFilter
	// database.Callback().Update().Before("gorm:update").Register("gorm:ignoreSoftDeleteItems", deleteFilter)
	// database.Callback().Query().Before("gorm:query").Register("gorm:ignoreSoftDeleteItems", deleteFilter)

	// 注意：所有表的迁移现在统一由 migrations 包管理
	// 迁移步骤在 pay/migrations_register.go 中注册
	// 迁移执行在应用启动时通过 migrations.GetRegistry().ExecuteAll() 统一调用

	return nil
}

// Close ...
func Close() error {
	if database == nil {
		log.Info("database is close")
	}
	db, err := database.DB()
	if err != nil {
		return err
	}
	db.Close()
	return nil
}

type Base struct {
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`
}

type IDBase struct {
	ID uint `gorm:"primaryKey;column:id;autoIncrement" json:"id"`
	Base
}

func DataBase() *gorm.DB {
	if database == nil {
		log.Panic("database connector not init")
		return nil
	}
	return database
}

func createOp(db *gorm.DB) {
	now := time.Now()
	fmt.Println("craete:", now.String())
	db.Set("create_at = ?", now).Set("update_at = ?", now)
}

func updateOp(db *gorm.DB) {
	now := time.Now()
	fmt.Println("update:", now.String())
	db.Set("update_at = ?", now)
}

func deleteFilter(db *gorm.DB) {
	db.Where("deleted = ?", 0)
}
