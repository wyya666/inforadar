export type Provider = 'deepseek' | 'zhipu' | 'bailian'
export type SearchProvider = 'zhipu' | 'bailian'

export interface RadarSettings {
  modelProvider: Provider
  modelId: string
  searchProvider: SearchProvider
  intervalHours: 1 | 6 | 12 | 24
  configuredKeys: Record<Provider, boolean>
  saved: boolean
}

export interface Topic {
  id: number
  description: string
  status: 'active' | 'paused'
  searchQuery?: string
  nextRunAt: string | null
  lastRunAt: string | null
  lastError?: string
  scanning: boolean
  waitingForConfig: boolean
  createdAt: string
}

export interface FeedItem {
  id: number
  topicId: number
  topicDescription: string
  url: string
  title: string
  snippet?: string
  summary?: string
  publishedAt: string | null
  firstSeenAt: string
}

export interface Feed { items: FeedItem[]; nextCursor: number | null }
export interface Run { id: number; topicId: number; status: string; resultCount: number; errorMessage?: string; startedAt: string; finishedAt: string | null }

export const providerNames: Record<Provider, string> = {
  deepseek: 'DeepSeek', zhipu: '智谱', bailian: '阿里百炼',
}

export const defaultModels: Record<Provider, string> = {
  deepseek: 'deepseek-flash', zhipu: 'glm-5.3-flash', bailian: 'qwen-plus',
}
