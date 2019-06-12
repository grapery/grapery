package models

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
	"time"
)

var database *gorm.DB

// Init
func Init() error {
	var err error
	if database != nil {
		log.Warn("database already init")
		return nil
	}
	//gorm.Open("mysql", "user:password@/dbname?charset=utf8&parseTime=True&loc=Local""mysql", "user:password@/dbname?charset=utf8&parseTime=True&loc=Local""mysql", "user:password@/dbname?charset=utf8&parseTime=True&loc=Local")
	database, err = gorm.Open("", "")
	if err != nil {
		log.Errorf("create dataabse failed  : [%s]", err.Error())
		return err
	}
	database.AutoMigrate(&User{})
	database.AutoMigrate(&Auth{})
	database.AutoMigrate(&Active{})
	database.AutoMigrate(&Event{})
	database.AutoMigrate(&Group{})
	database.AutoMigrate(&Profile{})
	database.AutoMigrate(&Project{})
	//database.AutoMigrate(&U)
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
	ID        uint      `gorm:"primary_key" json:"id,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	Deleted   bool
	DeletedAt *time.Time `sql:"index"`
}
