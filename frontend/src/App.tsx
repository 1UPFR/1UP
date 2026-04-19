import React, { useState, useEffect, useCallback } from 'react'
import { BrowserOpenURL } from '../wailsjs/runtime/runtime'
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

  return (
    <div className="layout">
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
        {update && (
          <div style={{
            position: 'absolute', inset: 0, zIndex: 9999,
            background: 'rgba(13,17,23,0.95)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
          }}>
            <div style={{ textAlign: 'center', maxWidth: 400 }}>
              <div style={{ fontSize: 48, marginBottom: 16 }}>&#9888;</div>
              <h2 style={{ fontSize: 20, fontWeight: 800, marginBottom: 8 }}>Mise a jour requise</h2>
              <p className="text-secondary" style={{ marginBottom: 16, lineHeight: 1.6 }}>
                La version <strong>v{update.version}</strong> est disponible.<br />
                Vous utilisez la version <strong>v{version}</strong>.
              </p>
              <p className="text-muted text-sm" style={{ marginBottom: 24 }}>
                Veuillez mettre a jour pour continuer a utiliser 1UP.
              </p>
              <button
                className="btn btn-primary btn-lg"
                style={{ justifyContent: 'center', width: '100%' }}
                onClick={() => BrowserOpenURL(update.url)}
              >
                Telecharger v{update.version}
              </button>
            </div>
          </div>
        )}
        <div style={{ display: page === 'upload' ? undefined : 'none', height: '100%' }}>
          <UploadPage addLog={addLog} logs={logs} />
        </div>
        <div style={{ display: page === 'manual' ? undefined : 'none', height: '100%' }}>
          <ManualPage />
        </div>
        <div style={{ display: page === 'history' ? undefined : 'none', height: '100%' }}>
          <HistoryPage visible={page === 'history'} />
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
