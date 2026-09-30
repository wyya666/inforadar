package repositoryinterface

import (
	"context"
	radarmodel "inforadar/internal/api/model/radar"
	"time"
)

type RadarRepository interface {
	SettingsByUser(context.Context, uint64) (radarmodel.Settings, error)
	SaveSettings(context.Context, *radarmodel.Settings) error
	CredentialByUser(context.Context, uint64, string) (radarmodel.Credential, error)
	CredentialStates(context.Context, uint64) (map[string]bool, error)
	SaveCredential(context.Context, *radarmodel.Credential) error
	DeleteCredential(context.Context, uint64, string) error
	WakeUserTopics(context.Context, uint64, time.Time) error
	SleepUserTopics(context.Context, uint64) error
	RescheduleUserTopics(context.Context, uint64, time.Time, time.Time) error
	ListTopics(context.Context, uint64) ([]radarmodel.Topic, error)
	TopicByID(context.Context, uint64, uint64) (radarmodel.Topic, error)
	CreateTopic(context.Context, *radarmodel.Topic) error
	UpdateTopic(context.Context, *radarmodel.Topic) error
	DeleteTopic(context.Context, uint64, uint64) error
	QueueManualScan(context.Context, uint64, uint64, time.Time) error
	ClaimDue(context.Context, time.Time) (radarmodel.Topic, radarmodel.Run, error)
	SaveQuery(context.Context, uint64, string) error
	ExistingURLHashes(context.Context, uint64, []string) (map[string]bool, error)
	SaveCandidates(context.Context, []radarmodel.Candidate) error
	FinishRun(context.Context, radarmodel.Topic, radarmodel.Run, string, uint, string, *time.Time, uint8) error
	ListRuns(context.Context, uint64, uint64, int) ([]radarmodel.Run, error)
	ListFeed(context.Context, uint64, uint64, uint64, int) ([]radarmodel.Candidate, map[uint64]string, error)
}
