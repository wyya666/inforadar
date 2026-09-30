import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'

export function AuthShell({ title, description, children }: { title: string; description: string; children: ReactNode }) {
  return <div className="auth-scene">
    <aside className="auth-story">
      <Link to="/" className="brand brand-light"><span className="brand-symbol"><span /></span><span>InfoRadar</span></Link>
      <div className="story-body">
        <div className="story-eyebrow"><span className="status-dot" /> SIGNAL OVER NOISE</div>
        <h2>重要的变化，<br /><em>不再错过。</em></h2>
        <p>让 InfoRadar 持续关注你在意的主题，把真正值得知道的新信息带到你面前。</p>
        <div className="story-signal" aria-hidden="true"><span /><i /></div>
      </div>
      <div className="story-bottom"><span>持续扫描</span><span>过滤噪声</span><span>主动交付</span></div>
    </aside>
    <section className="auth-form-area">
      <div className="auth-form-wrap">
        <Link to="/" className="brand brand-mobile"><span className="brand-symbol"><span /></span><span>InfoRadar</span></Link>
        <div className="form-heading"><h1>{title}</h1><p>{description}</p></div>
        {children}
      </div>
      <div className="auth-form-foot">© {new Date().getFullYear()} InfoRadar</div>
    </section>
  </div>
}
