import React, { useState, useEffect, useCallback, useRef } from 'react'
import UploadPage from './pages/UploadPage'
import SettingsPage from './pages/SettingsPage'
import JournalPage from './pages/JournalPage'
import HistoryPage from './pages/HistoryPage'
import ManualPage from './pages/ManualPage'
import { GetAppVersion, CheckUpdate } from '../wailsjs/go/main/App'
import logo from './assets/logo.png'

type Page = 'upload' | 'manual' | 'history' | 'settings' | 'journal'

export default function App() {
  const [page, setPage] = useState<Page>('upload')
  const [version, setVersion] = useState('')
  const [update, setUpdate] = useState<{ version: string; url: string } | null>(null)
  const [logs, setLogs] = useState<string[]>([])

  const addLog = useCallback((msg: string) => {
    const now = new Date().toLocaleTimeString()
    setLogs(prev => {
      const next = [...prev, `[${now}] ${msg}`]
      return next.length > 500 ? next.slice(-500) : next
    })
  }, [])

  useEffect(() => {
    GetAppVersion().then(v => setVersion(v)).catch(() => {})
    CheckUpdate()
      .then(info => { if (info.available) setUpdate({ version: info.latest, url: info.url }) })
      .catch(() => {})
  }, [])

  // Context menu
  const [ctxMenu, setCtxMenu] = useState<{ x: number; y: number; hasSelection: boolean } | null>(null)

  useEffect(() => {
    const handleContext = (e: MouseEvent) => {
      const target = e.target as HTMLElement
      if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable) {
        e.preventDefault()
        const sel = window.getSelection()
        setCtxMenu({ x: e.clientX, y: e.clientY, hasSelection: !!(sel && sel.toString()) })
      }
    }
    const handleClick = () => setCtxMenu(null)
    document.addEventListener('contextmenu', handleContext)
    document.addEventListener('click', handleClick)
    return () => {
      document.removeEventListener('contextmenu', handleContext)
      document.removeEventListener('click', handleClick)
    }
  }, [])

  const ctxAction = (action: string) => {
    document.execCommand(action)
    setCtxMenu(null)
  }

  return (
    <div className="layout">
      {ctxMenu && (
        <div className="ctx-menu" style={{ left: ctxMenu.x, top: ctxMenu.y }}>
          <button className="ctx-menu-item" disabled={!ctxMenu.hasSelection} onClick={() => ctxAction('cut')}>Couper</button>
          <button className="ctx-menu-item" disabled={!ctxMenu.hasSelection} onClick={() => ctxAction('copy')}>Copier</button>
          <button className="ctx-menu-item" onClick={() => ctxAction('paste')}>Coller</button>
          <div className="ctx-menu-sep" />
          <button className="ctx-menu-item" onClick={() => { document.execCommand('selectAll'); setCtxMenu(null) }}>Tout selectionner</button>
        </div>
      )}
      <aside className="sidebar">
        <div className="sidebar-logo">
          <img src={logo} alt="1UP" className="logo-img" />
        </div>

        <nav className="sidebar-nav">
          <div className="nav-section-label">NAVIGATION</div>
          <button
            className={`nav-item ${page === 'upload' ? 'active' : ''}`}
            onClick={() => setPage('upload')}
          >
            <span>&#9650;</span>
            <span>Upload</span>
          </button>
          <button
            className={`nav-item ${page === 'manual' ? 'active' : ''}`}
            onClick={() => setPage('manual')}
          >
            <span>&#9998;</span>
            <span>Manuel</span>
          </button>
          <button
            className={`nav-item ${page === 'history' ? 'active' : ''}`}
            onClick={() => setPage('history')}
          >
            <span>&#9201;</span>
            <span>Historique</span>
          </button>
          <button
            className={`nav-item ${page === 'settings' ? 'active' : ''}`}
            onClick={() => setPage('settings')}
          >
            <span>&#9881;</span>
            <span>Reglages</span>
          </button>
          <button
            className={`nav-item ${page === 'journal' ? 'active' : ''}`}
            onClick={() => setPage('journal')}
          >
            <span>&#9776;</span>
            <span>Journal</span>
          </button>
        </nav>

        <div className="sidebar-footer">
          <span className="text-muted text-xs">1UP v{version}</span>
          {update && (
            <a
              href={update.url}
              target="_blank"
              rel="noopener noreferrer"
              className="update-badge"
            >
              &#8593; v{update.version}
            </a>
          )}
        </div>
      </aside>

      <main className="main-content">
        <div style={{ display: page === 'upload' ? undefined : 'none', height: '100%' }}>
          <UploadPage addLog={addLog} logs={logs} />
        </div>
        <div style={{ display: page === 'manual' ? undefined : 'none', height: '100%' }}>
          <ManualPage />
        </div>
        <div style={{ display: page === 'history' ? undefined : 'none', height: '100%' }}>
          <HistoryPage />
        </div>
        <div style={{ display: page === 'settings' ? undefined : 'none', height: '100%' }}>
          <SettingsPage />
        </div>
        <div style={{ display: page === 'journal' ? undefined : 'none', height: '100%' }}>
          <JournalPage logs={logs} onClear={() => setLogs([])} />
        </div>
      </main>
    </div>
  )
}
