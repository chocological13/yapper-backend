package cronjob

import (
	"context"
	"os"
)

// scheduleMediaCleanupJob schedules the orphaned media cleanup job
func (s *scheduler) scheduleMediaCleanupJob() {
	ctx := context.Background()

	schedule := os.Getenv("CLEANUP_CRON_SCHEDULE")
	if schedule == "" {
		s.logger.Warn("CLEANUP_CRON_SCHEDULE is not set, using default schedule '0 */6 * * *'")
		schedule = "0 */6 * * *"
	}

	_, err := s.cron.AddFunc(schedule, func() {
		s.logger.Info("starting orphaned media cleanup job...")
		if err := s.mediaService.CleanUpOrphanedMedia(ctx); err != nil {
			s.logger.Error("media cleanup job failed", "error", err)
		} else {
			s.logger.Info("media cleanup job completed successfully.")
		}
	})
	if err != nil {
		s.logger.Error("failed to schedule media cleanup job", "error", err)
	}
}

func (s *scheduler) scheduleSoftDeletedMediaCleanupJob() {
	ctx := context.Background()

	schedule := os.Getenv("DAILY_CRON_SCHEDULE")
	if schedule == "" {
		s.logger.Warn("DAILY_CRON_SCHEDULE is not set, using default schedule '0 0 * * *'")
		schedule = "0 0 * * *"
	}

	_, err := s.cron.AddFunc(schedule, func() {
		s.logger.Info("starting soft deleted media cleanup job...")
		if err := s.mediaService.HardDeleteSoftDeletedMedia(ctx); err != nil {
			s.logger.Error("media hard delete job failed", "error", err)
		} else {
			s.logger.Info("media hard delete job completed successfully.")
		}
	})
	if err != nil {
		s.logger.Error("failed to schedule media hard delete job", "error", err)
	}
}
