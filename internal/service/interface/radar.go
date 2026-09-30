package serviceinterface

import (
	"context"
	radarmodel "inforadar/internal/api/model/radar"
	radarvo "inforadar/internal/api/vo/radar"
	"time"
)

type ModelClient interface {
	Complete(context.Context, string, string, string, string) (string, error)
}

type SearchClient interface {
	Search(context.Context, string, string, string) ([]radarmodel.SearchResult, error)
}

type RadarService interface {
	Settings(context.Context, uint64) (radarvo.Settings, error)
	SaveSettings(context.Context, uint64, string, string, string, uint8) (radarvo.Settings, error)
	SaveCredential(context.Context, uint64, string, string) error
	DeleteCredential(context.Context, uint64, string) error
	ListTopics(context.Context, uint64) ([]radarvo.Topic, error)
	CreateTopic(context.Context, uint64, string) (radarvo.Topic, error)
	UpdateTopic(context.Context, uint64, uint64, *string, *string) (radarvo.Topic, error)
	DeleteTopic(context.Context, uint64, uint64) error
	ManualScan(context.Context, uint64, uint64) error
	ListRuns(context.Context, uint64, uint64) ([]radarvo.Run, error)
	ListFeed(context.Context, uint64, uint64, uint64, int) (radarvo.Feed, error)
	RunDue(context.Context, time.Time) (bool, error)
	Wakeups() <-chan struct{}
}
