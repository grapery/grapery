package service

import (
	"github.com/gin-gonic/gin"
	"github.com/grapery/grapery/config"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	Stop chan struct{}
}

func NewService() *Service {
	return &Service{
		Stop: make(chan struct{}),
	}
}

func (s *Service) Run(cfg *config.Config) error {
	app := gin.Default()
	log.Info(app.GET)
	return nil
}
