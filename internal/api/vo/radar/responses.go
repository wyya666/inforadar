package radar

import "time"

type Settings struct {
	ModelProvider  string          `json:"modelProvider"`
	ModelID        string          `json:"modelId"`
	SearchProvider string          `json:"searchProvider"`
	IntervalHours  uint8           `json:"intervalHours"`
	ConfiguredKeys map[string]bool `json:"configuredKeys"`
	Saved          bool            `json:"saved"`
}

type Topic struct {
	ID               uint64     `json:"id"`
	Description      string     `json:"description"`
	Status           string     `json:"status"`
	SearchQuery      string     `json:"searchQuery,omitempty"`
	NextRunAt        *time.Time `json:"nextRunAt"`
	LastRunAt        *time.Time `json:"lastRunAt"`
	LastError        string     `json:"lastError,omitempty"`
	Scanning         bool       `json:"scanning"`
	WaitingForConfig bool       `json:"waitingForConfig"`
	CreatedAt        time.Time  `json:"createdAt"`
}

type Run struct {
	ID           uint64     `json:"id"`
	TopicID      uint64     `json:"topicId"`
	Status       string     `json:"status"`
	ResultCount  uint       `json:"resultCount"`
	ErrorMessage string     `json:"errorMessage,omitempty"`
	StartedAt    time.Time  `json:"startedAt"`
	FinishedAt   *time.Time `json:"finishedAt"`
}

type FeedItem struct {
	ID               uint64     `json:"id"`
	TopicID          uint64     `json:"topicId"`
	TopicDescription string     `json:"topicDescription"`
	URL              string     `json:"url"`
	Title            string     `json:"title"`
	Snippet          string     `json:"snippet,omitempty"`
	Summary          string     `json:"summary,omitempty"`
	PublishedAt      *time.Time `json:"publishedAt"`
	FirstSeenAt      time.Time  `json:"firstSeenAt"`
}

type Feed struct {
	Items      []FeedItem `json:"items"`
	NextCursor *uint64    `json:"nextCursor"`
}
