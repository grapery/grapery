package models

import (
	"context"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
)

var database *gorm.DB

// Init ...
func Init(uname, pwd, db string) error {
	var err error
	if database != nil {
		log.Warn("database already init")
		return nil
	}
	sqldbUrl := fmt.Sprintf("%s:%s@(localhost:3306)/%s?charset=utf8&parseTime=True&loc=Local", uname, pwd, db)
	println("mysql :", sqldbUrl)
	database, err = gorm.Open("mysql", sqldbUrl)
	if err != nil {
		log.Errorf("create dataabse failed  : [%s]", err.Error())
		return err
	}
	database.AutoMigrate(&User{})
	database.AutoMigrate(&Auth{})
	database.AutoMigrate(&Active{})
	database.AutoMigrate(&Group{})
	database.AutoMigrate(&UserProfile{})
	database.AutoMigrate(&Project{})
	return nil
}

// Close ...
func Close() error {
	if database == nil {
		log.Info("database is close")
	}
	return database.Close()
}

type Base struct {
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	Deleted   bool      `json:"deleted,omitempty" gorm:"index"`
}

type IDBase struct {
	ID uint `gorm:"primary_key" json:"id,omitempty"`
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

func NewRepository(ctx context.Context, userID uint64) *Repository {
	return &Repository{
		Ctx:    ctx,
		UserID: userID,
		db:     DataBase(),
	}
}

func DataBase() *gorm.DB {
	if database == nil {
		log.Warn("database connector not init")
		return nil
	}
	return database
}
