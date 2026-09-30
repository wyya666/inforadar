import { useCallback, useEffect, useState } from 'react'
import { Alert, Button, Empty, Form, Input, Modal, Popconfirm, Select, Skeleton, Tag, Typography, message } from 'antd'
import { Link, useLocation } from 'react-router-dom'
import { useAuth } from '../auth/AuthProvider'
import type { Feed, FeedItem, RadarSettings, Run, Topic } from '../radar'
import { AppShell } from './AppShell'

const dateTime = (value: string | null | undefined) => value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '尚未扫描'

function statusOf(topic: Topic) {
  if (topic.status === 'paused') return { text: '已暂停', kind: 'muted' }
  if (topic.scanning) return { text: '扫描中', kind: 'live' }
  if (topic.waitingForConfig) return { text: '待配置', kind: 'warn' }
  if (topic.lastError) return { text: '扫描异常', kind: 'warn' }
  if (!topic.lastRunAt) return { text: '等待首次扫描', kind: 'muted' }
  return { text: '监控中', kind: 'live' }
}

function sourceHost(url: string) {
  try { return new URL(url).hostname }
  catch { return '来源网页' }
}

export function HomePage() {
  const { request } = useAuth()
  const location = useLocation()
  const [messageApi, contextHolder] = message.useMessage()
  const [settings, setSettings] = useState<RadarSettings | null>(null)
  const [topics, setTopics] = useState<Topic[]>([])
  const [feed, setFeed] = useState<FeedItem[]>([])
  const [cursor, setCursor] = useState<number | null>(null)
  const [topicFilter, setTopicFilter] = useState<number>(0)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [editTopic, setEditTopic] = useState<Topic | null>(null)
  const [editValue, setEditValue] = useState('')
  const [runsTopic, setRunsTopic] = useState<Topic | null>(null)
  const [runs, setRuns] = useState<Run[]>([])
  const [form] = Form.useForm<{ description: string }>()

  const reload = useCallback(async (filter = topicFilter) => {
    const [nextSettings, nextTopics, nextFeed] = await Promise.all([
      request<RadarSettings>('/radar/settings'),
      request<Topic[]>('/radar/topics'),
      request<Feed>(`/radar/feed?limit=20${filter ? `&topicId=${filter}` : ''}`),
    ])
    setSettings(nextSettings); setTopics(nextTopics); setFeed(nextFeed.items); setCursor(nextFeed.nextCursor)
  }, [request, topicFilter])

  useEffect(() => {
    let active = true
    reload().catch((caught) => { if (active) setError(caught instanceof Error ? caught.message : '加载失败') })
      .finally(() => { if (active) setLoading(false) })
    const timer = window.setInterval(() => { if (document.visibilityState === 'visible') reload().catch(() => {}) }, 15000)
    return () => { active = false; window.clearInterval(timer) }
  }, [reload])

  useEffect(() => {
    if (location.hash === '#topics' || location.hash === '#signals') {
      document.getElementById(location.hash.slice(1))?.scrollIntoView({ block: 'start' })
    }
  }, [location.hash])

  async function createTopic(values: { description: string }) {
    setBusy(true); setError(null)
    try {
      await request<Topic>('/radar/topics', { method: 'POST', body: JSON.stringify(values) })
      form.resetFields(); await reload(); messageApi.success(configured ? '主题已添加，系统将开始扫描' : '主题已添加，配置服务后将自动扫描')
    } catch (caught) { setError(caught instanceof Error ? caught.message : '创建失败') }
    finally { setBusy(false) }
  }

  async function action(topic: Topic, method: string, path: string, body?: object, success = '已更新'): Promise<boolean> {
    setBusy(true)
    try {
      await request(`/radar/topics/${topic.id}${path}`, { method, ...(body ? { body: JSON.stringify(body) } : {}) })
      await reload(); messageApi.success(success); return true
    } catch (caught) { messageApi.error(caught instanceof Error ? caught.message : '操作失败'); return false }
    finally { setBusy(false) }
  }

  async function saveEdit() {
    if (!editTopic || editValue.trim().length < 3) return
    if (await action(editTopic, 'PATCH', '', { description: editValue.trim() }, '主题已更新')) setEditTopic(null)
  }

  async function changeFilter(value: number) {
    setTopicFilter(value); setLoading(true)
    try { await reload(value) } catch (caught) { setError(caught instanceof Error ? caught.message : '加载失败') }
    finally { setLoading(false) }
  }

  async function loadMore() {
    if (!cursor) return
    setBusy(true)
    try {
      const next = await request<Feed>(`/radar/feed?limit=20&before=${cursor}${topicFilter ? `&topicId=${topicFilter}` : ''}`)
      setFeed((current) => [...current, ...next.items]); setCursor(next.nextCursor)
    } catch (caught) { messageApi.error(caught instanceof Error ? caught.message : '加载失败') }
    finally { setBusy(false) }
  }

  async function showRuns(topic: Topic) {
    setRunsTopic(topic); setRuns([])
    try { setRuns(await request<Run[]>(`/radar/topics/${topic.id}/runs`)) }
    catch (caught) { messageApi.error(caught instanceof Error ? caught.message : '加载扫描记录失败') }
  }

  const configured = !!settings?.saved && !!settings.configuredKeys[settings.modelProvider] && !!settings.configuredKeys[settings.searchProvider]
  const activeCount = topics.filter((topic) => topic.status === 'active' && !topic.waitingForConfig).length

  return <AppShell>{contextHolder}<main className="radar-main">
    <section className="radar-intro" aria-label="运行总览">
      <div className="radar-intro-copy"><div className="radar-overline"><span className="status-dot" /> OBSERVATION / 信息雷达</div>
        <h1>让重要变化，<br /><em>浮出信息海面。</em></h1>
        <p>写下你正在关注的事。雷达会定时检索公开信息，只留下值得你阅读的新发现。</p>
        <div className="radar-hero-stats"><span><strong>{activeCount}</strong>运行主题</span><span><strong>{feed.length}</strong>当前列表发现</span><span><strong>{settings?.saved ? `${settings.intervalHours}h` : '—'}</strong>扫描间隔</span></div>
      </div>
      <div className="overview-orbit" aria-hidden="true"><span className="overview-ring ring-one" /><span className="overview-ring ring-two" /><span className="overview-ring ring-three" /><span className="overview-axis axis-horizontal" /><span className="overview-axis axis-vertical" /><span className="overview-core" /><span className="overview-point point-one" /><span className="overview-point point-two" /><small>MONITORING / LIVE</small></div>
    </section>

    {error && <Alert type="error" title={error} showIcon closable onClose={() => setError(null)} className="radar-alert" />}

    <div className="radar-layout">
      <section className="radar-topics" id="topics">
        <div className="radar-section-head"><div><span className="radar-small-label">01 / WATCHLIST</span><h2>我的监控主题</h2></div><span className="radar-count">{topics.length} 个</span></div>
        <div className="topic-compose">
          <Form form={form} onFinish={createTopic} layout="vertical" requiredMark={false}>
            <Form.Item name="description" rules={[{ required: true, whitespace: true, message: '写下你关心的主题' }, { min: 3, message: '至少输入 3 个字符' }, { max: 1000, message: '最多输入 1000 个字符' }]}>
              <Input.TextArea autoSize={{ minRows: 3, maxRows: 6 }} maxLength={1000} showCount placeholder="例如：关注国内 AI 搜索产品的新发布、融资与政策变化" />
            </Form.Item>
            <Button type="primary" htmlType="submit" loading={busy} className="radar-create">{configured ? '开始监控' : '创建主题'} <span aria-hidden="true">↗</span></Button>
          </Form>
          {!loading && !configured && <div className="topic-compose-hint"><span className="hint-dot" aria-hidden="true" /><span>服务尚未配置，主题创建后会等待扫描。</span><Link to="/radar/settings">去设置 ↗</Link></div>}
        </div>
        {loading ? <Skeleton active paragraph={{ rows: 4 }} /> : topics.length === 0 ? <div className="radar-blank"><Empty description="还没有监控主题。在上方写下一个关注方向。" /></div> : <div className="topic-list">
          {topics.map((topic) => { const status = statusOf(topic); return <article className="topic-card" key={topic.id}>
            <div className="topic-card-top"><span className={`radar-status status-${status.kind}`}><i />{status.text}</span><span className="topic-card-id">#{topic.id}</span></div>
            <h3>{topic.description}</h3>
            {topic.lastError && <p className="topic-error">{topic.lastError}</p>}
            <p className="topic-meta">上次扫描 {dateTime(topic.lastRunAt)}{topic.nextRunAt && <> · 下次 {dateTime(topic.nextRunAt)}</>}</p>
            <div className="topic-actions">
              <Button size="small" type="link" onClick={() => { setEditTopic(topic); setEditValue(topic.description) }} disabled={busy || topic.scanning}>编辑</Button>
              <Button size="small" type="link" onClick={() => action(topic, 'PATCH', '', { status: topic.status === 'active' ? 'paused' : 'active' }, topic.status === 'active' ? '已暂停监控' : '已恢复监控')} disabled={busy || topic.scanning}>{topic.status === 'active' ? '暂停' : '恢复'}</Button>
              <Button size="small" type="link" onClick={() => action(topic, 'POST', '/scan', undefined, '扫描已排队')} disabled={busy || topic.status !== 'active' || topic.scanning || !configured}>立即扫描</Button>
              <Button size="small" type="link" onClick={() => showRuns(topic)}>记录</Button>
              <Popconfirm title="删除这个主题及其发现？" description="删除后无法恢复。" onConfirm={() => action(topic, 'DELETE', '', undefined, '主题已删除')} okText="删除" cancelText="取消"><Button size="small" type="link" danger disabled={busy || topic.scanning}>删除</Button></Popconfirm>
            </div>
          </article> })}
        </div>}
      </section>

      <section className="radar-feed" id="signals">
        <div className="radar-section-head"><div><span className="radar-small-label">02 / SIGNALS</span><h2>重要发现</h2></div><Select aria-label="筛选主题" value={topicFilter} onChange={changeFilter} style={{ minWidth: 145 }} options={[{ value: 0, label: '全部主题' }, ...topics.map((topic) => ({ value: topic.id, label: topic.description.slice(0, 24) }))]} /></div>
        {loading ? <Skeleton active paragraph={{ rows: 8 }} /> : feed.length === 0 ? <div className="radar-blank"><Empty description={topics.length ? '暂无重要发现。扫描完成后，新结果会出现在这里。' : '创建主题后，这里会展示值得关注的新发现。'} /></div> : <div className="signal-list">
          {feed.map((item) => <article className="signal-card" key={item.id}>
            <div className="signal-meta"><span className="signal-kicker">IMPORTANT SIGNAL</span><span>{item.topicDescription}</span><time dateTime={item.firstSeenAt}>{dateTime(item.firstSeenAt)}</time></div>
            <h3><a href={item.url} target="_blank" rel="noopener noreferrer">{item.title} <span aria-hidden="true">↗</span></a></h3>
            {item.summary && <p className="signal-summary">{item.summary}</p>}
            {item.snippet && <Typography.Paragraph ellipsis={{ rows: 3 }} className="signal-snippet">{item.snippet}</Typography.Paragraph>}
            <div className="signal-source"><span>{sourceHost(item.url)}</span>{item.publishedAt && <span>发布于 {dateTime(item.publishedAt)}</span>}<a href={item.url} target="_blank" rel="noopener noreferrer">查看来源 ↗</a></div>
          </article>)}
          {cursor && <Button block onClick={loadMore} loading={busy} className="load-more">加载更多发现</Button>}
        </div>}
      </section>
    </div>
    <Modal title="编辑监控主题" open={!!editTopic} onOk={saveEdit} onCancel={() => setEditTopic(null)} okText="保存" cancelText="取消" okButtonProps={{ disabled: editValue.trim().length < 3, loading: busy }}><Input.TextArea value={editValue} onChange={(event) => setEditValue(event.target.value)} maxLength={1000} showCount autoSize={{ minRows: 4, maxRows: 8 }} /><p className="edit-hint">修改描述后会重新生成搜索词，并清除该主题此前的发现。</p></Modal>
    <Modal title="扫描记录" open={!!runsTopic} onCancel={() => setRunsTopic(null)} footer={null}>
      {runs.length === 0 ? <Empty description="暂无扫描记录" /> : <div className="run-list">{runs.map((run) => <div className="run-row" key={run.id}><Tag color={run.status === 'succeeded' ? 'success' : run.status === 'running' ? 'processing' : run.status === 'skipped' ? 'warning' : 'error'}>{run.status === 'succeeded' ? '完成' : run.status === 'running' ? '运行中' : run.status === 'skipped' ? '未执行' : '失败'}</Tag><span>{dateTime(run.startedAt)}</span><span>{run.status === 'succeeded' ? `${run.resultCount} 条重要发现` : run.errorMessage || ''}</span></div>)}</div>}
    </Modal>
  </main></AppShell>
}
