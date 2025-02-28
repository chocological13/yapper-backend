package cronjob

import (
	"github.com/chocological13/yapper-backend/pkg/media"
	"github.com/robfig/cron/v3"
	"log/slog"
)

type Scheduler interface {
	Start()
	Stop()
}

type scheduler struct {
	cron         *cron.Cron
	mediaService media.MediaService
	logger       *slog.Logger
}

func NewScheduler(mediaService media.MediaService, logger *slog.Logger) Scheduler {
	return &scheduler{
		cron:         cron.New(),
		mediaService: mediaService,
		logger:       logger,
	}
}

func (s *scheduler) Start() {
	s.scheduleMediaCleanupJob()
	s.scheduleSoftDeletedMediaCleanupJob()
	s.cron.Start()
	s.logger.Info("cron scheduler started")
}

func (s *scheduler) Stop() {
	s.cron.Stop()
	s.logger.Info("cron scheduler stopped")
}
