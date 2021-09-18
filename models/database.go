package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	logger "gorm.io/gorm/logger"
)

var database *gorm.DB
var sqlDB *sql.DB

const (
	maxIdleConns    = 10
	maxOpenConns    = 20
	connMaxLifetime = 3600
)

// Init ...
func Init(uname, pwd, dbname string) error {
	var err error
	if database != nil {
		log.Warn("database already init")
		return nil
	}
	newLogger := logger.New(
		log.StandardLogger(),
		logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Info,
			Colorful:      true,
		},
	)
	sqldbUrl := fmt.Sprintf("%s:%s@(localhost:3306)/%s?charset=utf8&parseTime=True&loc=Local", uname, pwd, dbname)
	println("mysql :", sqldbUrl)

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
	/*callback: https://github.com/go-gorm/gorm/blob/master/callbacks/callbacks.go*/
	database.Callback().Create().Before("gorm:create").Register("gorm:update_ctime_mtime", createOp)
	database.Callback().Update().Before("gorm:update").Register("gorm:update_mtime", updateOp)
	database.Callback().Update().Before("gorm:update").Register("gorm:ignoreSoftDeleteItems", deleteFilter)
	database.Callback().Query().Before("gorm:query").Register("gorm:ignoreSoftDeleteItems", deleteFilter)

	database.AutoMigrate(&User{})
	database.AutoMigrate(&Auth{})
	database.AutoMigrate(&Active{})
	database.AutoMigrate(&Group{})
	database.AutoMigrate(&Project{})
	database.AutoMigrate(&GroupMember{})
	database.AutoMigrate(&Item{})
	database.AutoMigrate(&LikeItem{})
	database.AutoMigrate(&Comment{})
	database.AutoMigrate(&Question{})
	database.AutoMigrate(&Team{})
	database.AutoMigrate(&TeamMemeber{})
	database.AutoMigrate(&Topic{})
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
	CreatedAt time.Time `gorm:"primary_key,column:create_at" json:"created_at,omitempty"`
	UpdatedAt time.Time `gorm:"primary_key,column:update_at" json:"updated_at,omitempty"`
	Deleted   bool      `gorm:"primary_key,column:deleted" json:"deleted,omitempty"`
}

type IDBase struct {
	ID uint `gorm:"primary_key,column:id" json:"id,omitempty"`
	Base
}

type Repository struct {
	Ctx    context.Context
	UserID uint64
	db     *gorm.DB
}

func (r *Repository) DB() *gorm.DB {
	return r.db
}

func NewRepository(ctx context.Context) *Repository {
	return &Repository{
		Ctx: ctx,
		db:  DataBase(),
	}
}

func DataBase() *gorm.DB {
	if database == nil {
		log.Panic("database connector not init")
		return nil
	}
	return database
}

func createOp(db *gorm.DB) {
	now := time.Now().Unix()
	db.Update("create_at = ?", now).Update("update_at = ?", now)
}

func updateOp(db *gorm.DB) {
	now := time.Now().Unix()
	db.Update("update_at = ?", now)
}

func deleteFilter(db *gorm.DB) {
	db.Where("deleted = ?", 0)
}
