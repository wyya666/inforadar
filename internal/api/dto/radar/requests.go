package radar

type SettingsRequest struct {
	ModelProvider  string `json:"modelProvider" binding:"required,oneof=deepseek zhipu bailian"`
	ModelID        string `json:"modelId" binding:"required,min=1,max=128"`
	SearchProvider string `json:"searchProvider" binding:"required,oneof=zhipu bailian"`
	IntervalHours  uint8  `json:"intervalHours" binding:"required,oneof=1 6 12 24"`
}

type CredentialRequest struct {
	APIKey string `json:"apiKey" binding:"required,min=8,max=2048"`
}

type TopicRequest struct {
	Description string `json:"description" binding:"required,min=3,max=1000"`
}

type TopicUpdateRequest struct {
	Description *string `json:"description" binding:"omitempty,min=3,max=1000"`
	Status      *string `json:"status" binding:"omitempty,oneof=active paused"`
}
