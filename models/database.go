package models

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
	sqldbUrl := fmt.Sprintf("%s:%s@(%s:3306)/%s?charset=utf8&parseTime=True&loc=Local", uname, pwd, address, dbname)
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
	database.Callback().Update().Before("gorm:update").Register("gorm:ignoreSoftDeleteItems", deleteFilter)
	database.Callback().Query().Before("gorm:query").Register("gorm:ignoreSoftDeleteItems", deleteFilter)

	database.AutoMigrate(&StoryItem{})
	database.AutoMigrate(&User{})
	database.AutoMigrate(&Auth{})
	database.AutoMigrate(&Active{})
	database.AutoMigrate(&Group{})
	database.AutoMigrate(&Project{})
	database.AutoMigrate(&GroupMember{})
	database.AutoMigrate(&LikeItem{})

	database.AutoMigrate(&WatchItem{})

	database.AutoMigrate(&UserProfile{})
	database.AutoMigrate(&ProjectProfile{})
	database.AutoMigrate(&GroupProfile{})

	database.AutoMigrate(&ProjectWatcher{})
	database.AutoMigrate(&Story{})
	database.AutoMigrate(&StoryBoard{})
	database.AutoMigrate(&StoryGen{})
	database.AutoMigrate(&Prompt{})
	database.AutoMigrate(&StoryBoardScene{})
	database.AutoMigrate(&StoryRole{})
	database.AutoMigrate(&ChatContext{})
	database.AutoMigrate(&ChatMessage{})
	database.AutoMigrate(&StoryBoardRole{})

	database.AutoMigrate(&Comment{})
	database.AutoMigrate(&CommentLike{})

	database.AutoMigrate(&Order{})
	database.AutoMigrate(&UserSession{})
	database.AutoMigrate(&LLMChatMsg{})
	database.AutoMigrate(&LLMMsgFeedback{})
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
	CreateAt time.Time `gorm:"column:create_at;autoCreateTime" json:"create_at,omitempty"`
	UpdateAt time.Time `gorm:"column:update_at;autoUpdateTime" json:"update_at,omitempty"`
	Deleted  bool      `gorm:"primary_key,column:deleted" json:"deleted,omitempty"`
}

type IDBase struct {
	ID uint `gorm:"primary_key,column:id" json:"id,omitempty"`
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
