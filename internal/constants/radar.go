package constants

import "time"

const (
	RadarRoute               = "/radar"
	ProviderDeepSeek         = "deepseek"
	ProviderZhipu            = "zhipu"
	ProviderBailian          = "bailian"
	DefaultDeepSeekModel     = "deepseek-flash"
	DefaultZhipuModel        = "glm-5.3-flash"
	DefaultBailianModel      = "qwen-plus"
	TopicActive              = "active"
	TopicPaused              = "paused"
	RunRunning               = "running"
	RunSucceeded             = "succeeded"
	RunFailed                = "failed"
	RunSkipped               = "skipped"
	RadarMaxCandidates       = 10
	RadarQueryMaxRunes       = 70
	RadarTopicMaxRunes       = 1000
	RadarManualCooldown      = 10 * time.Minute
	RadarLeaseDuration       = 5 * time.Minute
	RadarRunTimeout          = 4 * time.Minute
	RadarPollInterval        = time.Minute
	RadarSearchWindowDays    = 7
	RadarMaxRetries          = 3
	TableRadarSettings       = "radar_settings"
	TableProviderCredentials = "provider_credentials"
	TableMonitorTopics       = "monitor_topics"
	TableMonitorRuns         = "monitor_runs"
	TableMonitorCandidates   = "monitor_candidates"
	DeepSeekChatEndpoint     = "https://api.deepseek.com/chat/completions"
	ZhipuChatEndpoint        = "https://open.bigmodel.cn/api/paas/v4/chat/completions"
	ZhipuSearchEndpoint      = "https://open.bigmodel.cn/api/paas/v4/web_search"
	BailianChatEndpoint      = "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions"
	BailianSearchEndpoint    = "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation"
)
