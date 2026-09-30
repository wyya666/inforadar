package radar

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	radarmodel "inforadar/internal/api/model/radar"
	"inforadar/internal/common"
	"inforadar/internal/constants"
	repositoryinterface "inforadar/internal/repository/interface"
)

type Repository struct{ db *gorm.DB }

var _ repositoryinterface.RadarRepository = (*Repository)(nil)

func New(db *gorm.DB) repositoryinterface.RadarRepository { return &Repository{db: db} }

func translate(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return common.ErrNotFound
	}
	return err
}

func (r *Repository) SettingsByUser(ctx context.Context, userID uint64) (radarmodel.Settings, error) {
	var row radarmodel.Settings
	err := r.db.WithContext(ctx).First(&row, "user_id = ?", userID).Error
	return row, translate(err)
}

func (r *Repository) SaveSettings(ctx context.Context, row *radarmodel.Settings) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"model_provider", "model_id", "search_provider", "interval_hours", "updated_at"}),
	}).Create(row).Error
}

func (r *Repository) CredentialByUser(ctx context.Context, userID uint64, provider string) (radarmodel.Credential, error) {
	var row radarmodel.Credential
	err := r.db.WithContext(ctx).First(&row, "user_id = ? AND provider = ?", userID, provider).Error
	return row, translate(err)
}

func (r *Repository) CredentialStates(ctx context.Context, userID uint64) (map[string]bool, error) {
	var names []string
	err := r.db.WithContext(ctx).Model(&radarmodel.Credential{}).Where("user_id = ?", userID).Pluck("provider", &names).Error
	if err != nil {
		return nil, err
	}
	result := map[string]bool{constants.ProviderDeepSeek: false, constants.ProviderZhipu: false, constants.ProviderBailian: false}
	for _, name := range names {
		result[name] = true
	}
	return result, nil
}

func (r *Repository) SaveCredential(ctx context.Context, row *radarmodel.Credential) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "provider"}},
		DoUpdates: clause.AssignmentColumns([]string{"ciphertext", "nonce", "updated_at"}),
	}).Create(row).Error
}

func (r *Repository) DeleteCredential(ctx context.Context, userID uint64, provider string) error {
	return r.db.WithContext(ctx).Where("user_id = ? AND provider = ?", userID, provider).Delete(&radarmodel.Credential{}).Error
}

func (r *Repository) WakeUserTopics(ctx context.Context, userID uint64, when time.Time) error {
	return r.db.WithContext(ctx).Model(&radarmodel.Topic{}).Where("user_id = ? AND status = ? AND (lease_until IS NULL OR lease_until < ?)", userID, constants.TopicActive, when).
		Updates(map[string]any{"next_run_at": when, "retry_count": 0, "last_error": nil}).Error
}

func (r *Repository) SleepUserTopics(ctx context.Context, userID uint64) error {
	return r.db.WithContext(ctx).Model(&radarmodel.Topic{}).
		Where("user_id = ? AND status = ? AND lease_until IS NULL", userID, constants.TopicActive).
		Updates(map[string]any{"next_run_at": nil, "last_error": constants.MessageRadarUnconfigured}).Error
}

func (r *Repository) RescheduleUserTopics(ctx context.Context, userID uint64, now, next time.Time) error {
	return r.db.WithContext(ctx).Model(&radarmodel.Topic{}).
		Where("user_id = ? AND status = ? AND lease_until IS NULL", userID, constants.TopicActive).
		Update("next_run_at", gorm.Expr("CASE WHEN last_run_at IS NULL THEN ? ELSE ? END", now, next)).Error
}

func (r *Repository) ListTopics(ctx context.Context, userID uint64) ([]radarmodel.Topic, error) {
	var rows []radarmodel.Topic
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("id DESC").Find(&rows).Error
	return rows, err
}

func (r *Repository) TopicByID(ctx context.Context, userID, topicID uint64) (radarmodel.Topic, error) {
	var row radarmodel.Topic
	err := r.db.WithContext(ctx).First(&row, "id = ? AND user_id = ?", topicID, userID).Error
	return row, translate(err)
}

func (r *Repository) CreateTopic(ctx context.Context, row *radarmodel.Topic) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) UpdateTopic(ctx context.Context, row *radarmodel.Topic) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current radarmodel.Topic
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, "id = ? AND user_id = ?", row.ID, row.UserID).Error; err != nil {
			return translate(err)
		}
		if current.LeaseUntil != nil && current.LeaseUntil.After(time.Now().UTC()) {
			return common.ErrRadarBusy
		}
		if current.Description != row.Description {
			if err := tx.Where("user_id = ? AND topic_id = ?", row.UserID, row.ID).Delete(&radarmodel.Candidate{}).Error; err != nil {
				return err
			}
		}
		return tx.Model(&current).Updates(map[string]any{"description": row.Description, "search_query": row.SearchQuery, "status": row.Status, "next_run_at": row.NextRunAt, "retry_count": row.RetryCount, "last_error": row.LastError, "updated_at": row.UpdatedAt}).Error
	})
}

func (r *Repository) DeleteTopic(ctx context.Context, userID, topicID uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row radarmodel.Topic
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id = ? AND user_id = ?", topicID, userID).Error; err != nil {
			return translate(err)
		}
		if row.LeaseUntil != nil && row.LeaseUntil.After(time.Now().UTC()) {
			return common.ErrRadarBusy
		}
		if err := tx.Where("topic_id = ? AND user_id = ?", topicID, userID).Delete(&radarmodel.Candidate{}).Error; err != nil {
			return err
		}
		if err := tx.Where("topic_id = ? AND user_id = ?", topicID, userID).Delete(&radarmodel.Run{}).Error; err != nil {
			return err
		}
		return tx.Delete(&row).Error
	})
}

func (r *Repository) QueueManualScan(ctx context.Context, userID, topicID uint64, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row radarmodel.Topic
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id = ? AND user_id = ?", topicID, userID).Error; err != nil {
			return translate(err)
		}
		if row.Status != constants.TopicActive {
			return common.ErrInvalidInput
		}
		if row.LeaseUntil != nil && row.LeaseUntil.After(now) {
			return common.ErrRadarBusy
		}
		if row.LastManualAt != nil && row.LastManualAt.Add(constants.RadarManualCooldown).After(now) {
			return common.ErrRadarCooldown
		}
		return tx.Model(&row).Updates(map[string]any{"next_run_at": now, "last_manual_at": now, "retry_count": 0, "last_error": nil}).Error
	})
}

// ClaimDue locks one due topic. SKIP LOCKED lets multiple API instances cooperate.
func (r *Repository) ClaimDue(ctx context.Context, now time.Time) (radarmodel.Topic, radarmodel.Run, error) {
	var topic radarmodel.Topic
	var run radarmodel.Run
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ? AND ((next_run_at <= ? AND (lease_until IS NULL OR lease_until <= ?)) OR lease_until <= ?)", constants.TopicActive, now, now, now).
			Order("next_run_at ASC, id ASC").First(&topic).Error
		if err != nil {
			return translate(err)
		}
		if err := tx.Model(&radarmodel.Run{}).Where("topic_id = ? AND status = ?", topic.ID, constants.RunRunning).
			Updates(map[string]any{"status": constants.RunFailed, "error_message": "扫描中断，已重新排队", "finished_at": now}).Error; err != nil {
			return err
		}
		lease := now.Add(constants.RadarLeaseDuration).UTC().Truncate(time.Millisecond)
		if err := tx.Model(&topic).Updates(map[string]any{"lease_until": lease, "next_run_at": nil}).Error; err != nil {
			return err
		}
		topic.LeaseUntil = &lease
		run = radarmodel.Run{UserID: topic.UserID, TopicID: topic.ID, Status: constants.RunRunning, StartedAt: now}
		return tx.Create(&run).Error
	})
	return topic, run, err
}

func (r *Repository) SaveQuery(ctx context.Context, topicID uint64, query string) error {
	return r.db.WithContext(ctx).Model(&radarmodel.Topic{}).Where("id = ?", topicID).Update("search_query", query).Error
}

func (r *Repository) ExistingURLHashes(ctx context.Context, topicID uint64, hashes []string) (map[string]bool, error) {
	result := make(map[string]bool)
	if len(hashes) == 0 {
		return result, nil
	}
	var existing []string
	err := r.db.WithContext(ctx).Model(&radarmodel.Candidate{}).Where("topic_id = ? AND url_hash IN ?", topicID, hashes).Pluck("url_hash", &existing).Error
	for _, hash := range existing {
		result[hash] = true
	}
	return result, err
}

func (r *Repository) SaveCandidates(ctx context.Context, rows []radarmodel.Candidate) error {
	if len(rows) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error
}

func (r *Repository) FinishRun(ctx context.Context, topic radarmodel.Topic, run radarmodel.Run, status string, count uint, message string, next *time.Time, retries uint8) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var msg any
		if message != "" {
			msg = message
		}
		runUpdate := tx.Model(&radarmodel.Run{}).Where("id = ? AND status = ?", run.ID, constants.RunRunning).
			Updates(map[string]any{"status": status, "result_count": count, "error_message": msg, "finished_at": now})
		if runUpdate.Error != nil {
			return runUpdate.Error
		}
		if runUpdate.RowsAffected != 1 {
			return common.ErrRadarBusy
		}
		result := tx.Model(&radarmodel.Topic{}).Where("id = ? AND lease_until = ?", topic.ID, topic.LeaseUntil).
			Updates(map[string]any{"lease_until": nil, "last_run_at": now, "next_run_at": next, "retry_count": retries, "last_error": msg})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return common.ErrRadarBusy
		}
		return nil
	})
}

func (r *Repository) ListRuns(ctx context.Context, userID, topicID uint64, limit int) ([]radarmodel.Run, error) {
	if _, err := r.TopicByID(ctx, userID, topicID); err != nil {
		return nil, err
	}
	var rows []radarmodel.Run
	err := r.db.WithContext(ctx).Where("user_id = ? AND topic_id = ?", userID, topicID).Order("id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *Repository) ListFeed(ctx context.Context, userID, topicID, before uint64, limit int) ([]radarmodel.Candidate, map[uint64]string, error) {
	query := r.db.WithContext(ctx).Model(&radarmodel.Candidate{}).Where("user_id = ? AND important = 1", userID)
	if topicID != 0 {
		query = query.Where("topic_id = ?", topicID)
	}
	if before != 0 {
		query = query.Where("id < ?", before)
	}
	var rows []radarmodel.Candidate
	if err := query.Order("id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, nil, err
	}
	var topics []radarmodel.Topic
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&topics).Error; err != nil {
		return nil, nil, err
	}
	names := make(map[uint64]string, len(topics))
	for _, row := range topics {
		names[row.ID] = row.Description
	}
	return rows, names, nil
}
