package radar

import (
	"inforadar/internal/constants"
	"time"
)

type Settings struct {
	UserID         uint64 `gorm:"primaryKey"`
	ModelProvider  string
	ModelID        string `gorm:"column:model_id"`
	SearchProvider string
	IntervalHours  uint8
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (Settings) TableName() string { return constants.TableRadarSettings }

type Credential struct {
	UserID     uint64 `gorm:"primaryKey"`
	Provider   string `gorm:"primaryKey"`
	Ciphertext []byte
	Nonce      []byte
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (Credential) TableName() string { return constants.TableProviderCredentials }

type Topic struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement"`
	UserID       uint64
	Description  string
	SearchQuery  *string
	Status       string
	NextRunAt    *time.Time
	LastRunAt    *time.Time
	LastManualAt *time.Time
	LeaseUntil   *time.Time
	RetryCount   uint8
	LastError    *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (Topic) TableName() string { return constants.TableMonitorTopics }

type Run struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement"`
	UserID       uint64
	TopicID      uint64
	Status       string
	ResultCount  uint
	ErrorMessage *string
	StartedAt    time.Time
	FinishedAt   *time.Time
}

func (Run) TableName() string { return constants.TableMonitorRuns }

type Candidate struct {
	ID          uint64 `gorm:"primaryKey;autoIncrement"`
	UserID      uint64
	TopicID     uint64
	URLHash     string `gorm:"column:url_hash"`
	URL         string `gorm:"column:url"`
	Title       string
	Snippet     *string
	PublishedAt *time.Time
	Important   bool
	Summary     *string
	FirstSeenAt time.Time
}

func (Candidate) TableName() string { return constants.TableMonitorCandidates }

type SearchResult struct {
	URL         string
	Title       string
	Snippet     string
	PublishedAt *time.Time
}
