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

const (
	maxIdleConns    = 10
	maxOpenConns    = 20
	connMaxLifetime = 3600
)

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
	database.DB().SetMaxIdleConns(maxIdleConns)
	database.DB().SetMaxOpenConns(maxOpenConns)
	database.DB().SetConnMaxLifetime(connMaxLifetime)
	database.Debug()
	/*callback: https://github.com/go-gorm/gorm/blob/master/callbacks/callbacks.go*/
	database.Callback().Create().Before("gorm:create").Register("gorm:update_ctime_mtime", createCallback)
	database.Callback().Update().Before("gorm:update").Register("gorm:update_mtime", updateCallback)
	database.Callback().Update().Before("gorm:update").Register("gorm:ignoreSoftDeleteItems", ignoreSoftDeleteItems)
	database.Callback().Query().Before("gorm:query").Register("gorm:ignoreSoftDeleteItems", ignoreSoftDeleteItems)

	database.AutoMigrate(&User{})
	database.AutoMigrate(&Auth{})
	database.AutoMigrate(&Active{})
	database.AutoMigrate(&Group{})
	database.AutoMigrate(&Project{})
	database.AutoMigrate(&Item{})
	database.AutoMigrate(&ShareItem{})
	database.AutoMigrate(&LikeItem{})
	database.AutoMigrate(&Comment{})
	//database.AutoMigrate(&Question{})
	//database.SetLogger(log.New())
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

func createCallback(scope *gorm.Scope) {
	if !scope.HasError() {
		now := time.Now().Unix()

		if createdAtField, ok := scope.FieldByName("create_at"); ok {
			if createdAtField.IsBlank {
				_ = createdAtField.Set(now)
			}
		}
		if updatedAtField, ok := scope.FieldByName("update_at"); ok {
			if updatedAtField.IsBlank {
				_ = updatedAtField.Set(now)
			}
		}
	}
}

func updateCallback(scope *gorm.Scope) {
	if !scope.HasError() {
		now := time.Now().Unix()
		if updatedAtField, ok := scope.FieldByName("update_at"); ok {
			if updatedAtField.IsBlank {
				_ = updatedAtField.Set(now)
				_ = scope.SetColumn("update_at", now)
			}
		}
	}
}

func ignoreSoftDeleteItems(scope *gorm.Scope) {
	if !scope.HasError() {
		deletedTimeField, hasDeletedTimeField := scope.FieldByName("deleted")
		if !scope.Search.Unscoped && hasDeletedTimeField {
			scope.Search.Where(fmt.Sprintf("%s = ?", scope.Quote(deletedTimeField.DBName)), 0)
		}
	}
}
