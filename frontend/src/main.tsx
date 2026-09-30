import { createRoot } from 'react-dom/client'
import { App as AntApp, ConfigProvider } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import App from './App'
import './styles.css'

createRoot(document.getElementById('root')!).render(
  <ConfigProvider
    locale={zhCN}
    theme={{ token: { colorPrimary: '#168a81', colorInfo: '#168a81', colorSuccess: '#168a81', colorWarning: '#c77c4a', colorText: '#243f46', colorBgLayout: '#f4f8f6', colorBorder: '#dbe7e2', borderRadius: 8, fontFamily: '"PingFang SC", "Noto Sans CJK SC", "Microsoft YaHei", Arial, sans-serif' } }}
  >
    <AntApp><App /></AntApp>
  </ConfigProvider>,
)
