package models

import (
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
)

var database *gorm.DB

// Init
func Init(uname, pwd, db string) error {
	var err error
	if database != nil {
		log.Warn("database already init")
		return nil
	}
	//gorm.Open("mysql", "user:password@/dbname?charset=utf8&parseTime=True&loc=Local")
	sqldbUrl := fmt.Sprintf("%s:%s@/%s?charset=utf8&parseTime=True&loc=Local", uname, pwd, db)
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

// Close
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
