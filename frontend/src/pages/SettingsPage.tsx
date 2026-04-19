import React, { useState, useEffect, useCallback } from 'react'
import { GetConfig, SaveConfig, SelectDirectory, TestUsenet } from '../../wailsjs/go/main/App'

interface Config {
  nyuu: { host: string; port: number; user: string; password: string; connections: number; group: string; ssl: boolean; extra_args: string }
  parpar: { slice_size: string; memory: string; threads: number; redundancy: string; extra_args: string }
  api: { url: string; apikey: string; enabled: boolean }
  output_dir: string
}

export default function SettingsPage() {
  const [config, setConfig] = useState<Config | null>(null)
  const [saved, setSaved] = useState(false)
  const [showPassword, setShowPassword] = useState(false)
  const [testing, setTesting] = useState(false)
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null)

  useEffect(() => {
    GetConfig().then((c: any) => setConfig(c)).catch(console.error)
  }, [])

  const save = useCallback(async (cfg: any) => {
    try {
      await SaveConfig(cfg as any)
      setSaved(true)
      setTimeout(() => setSaved(false), 1500)
    } catch (e) {
      console.error('Erreur sauvegarde:', e)
    }
  }, [])

  const update = useCallback((path: string, value: any) => {
    setConfig(prev => {
      if (!prev) return prev
      const next = JSON.parse(JSON.stringify(prev))
      const parts = path.split('.')
      let obj = next
      for (let i = 0; i < parts.length - 1; i++) obj = obj[parts[i]]
      obj[parts[parts.length - 1]] = value
      save(next)
      return next
    })
  }, [save])

  if (!config) return <div className="spinner" style={{ margin: '40px auto', display: 'block' }} />

  const configPath = (typeof navigator !== 'undefined' && navigator.platform?.includes('Win'))
    ? '%USERPROFILE%\\.config\\1up\\config.json'
    : '~/.config/1up/config.json'

  return (
    <div>
      <div className="page-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <div>
          <h1 className="page-title">Reglages</h1>
          <p className="text-muted text-xs" style={{ marginTop: 4, fontFamily: 'monospace' }}>{configPath}</p>
        </div>
        {saved && <span className="badge badge-success">Sauvegarde</span>}
      </div>

      {/* Nyuu */}
      <div className="card" style={{ marginBottom: 12 }}>
        <div className="card-header" style={{ marginBottom: 10 }}>
          <span className="card-title" style={{ fontSize: 13 }}>Serveur Usenet (Nyuu)</span>
          <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
            {testResult && (
              <span style={{ fontSize: 12, fontWeight: 600, color: testResult.success ? 'var(--color-success)' : 'var(--color-danger)' }}>
                {testResult.message}
              </span>
            )}
            <button
              className="btn btn-primary btn-sm"
              disabled={testing || !config.nyuu.host}
              onClick={async () => {
                setTesting(true); setTestResult(null)
                try { const r = await TestUsenet(); setTestResult(r as any) } catch (e) { setTestResult({ success: false, message: String(e) }) }
                setTesting(false)
              }}
            >
              {testing ? <span className="spinner" style={{ width: 12, height: 12, borderWidth: 2 }} /> : 'Tester'}
            </button>
          </div>
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr 80px 1fr', gap: 12 }}>
          <Field label="Hote" value={config.nyuu.host} onChange={v => update('nyuu.host', v)} placeholder="news.example.com" />
          <Field label="Port" value={String(config.nyuu.port)} onChange={v => update('nyuu.port', parseInt(v) || 563)} type="number" />
          <div className="form-group">
            <label className="label">SSL</label>
            <label className="toggle" style={{ marginTop: 6 }}>
              <input type="checkbox" checked={config.nyuu.ssl} onChange={e => update('nyuu.ssl', e.target.checked)} />
              <span className="toggle-slider"></span>
            </label>
          </div>
          <Field label="Connexions" value={String(config.nyuu.connections)} onChange={v => update('nyuu.connections', parseInt(v) || 20)} type="number" />
        </div>
        <div className="grid-3">
          <Field label="Utilisateur" value={config.nyuu.user} onChange={v => update('nyuu.user', v)} />
          <div className="form-group">
            <label className="label">Mot de passe</label>
            <div style={{ display: 'flex', gap: 4 }}>
              <input
                className="input"
                type={showPassword ? 'text' : 'password'}
                value={config.nyuu.password}
                onChange={e => update('nyuu.password', e.target.value)}
              />
              <button
                className="btn btn-ghost btn-sm"
                onClick={() => setShowPassword(v => !v)}
                title={showPassword ? 'Masquer' : 'Afficher'}
                style={{ fontSize: 16, padding: '4px 8px', flexShrink: 0 }}
              >
                {showPassword ? '\u{1F441}' : '\u{1F441}\u{200D}\u{1F5E8}'}
              </button>
            </div>
          </div>
          <Field label="Groupe" value={config.nyuu.group} onChange={v => update('nyuu.group', v)} />
        </div>
      </div>

      {/* ParPar */}
      <div className="card" style={{ marginBottom: 12 }}>
        <div className="card-header" style={{ marginBottom: 10 }}>
          <span className="card-title" style={{ fontSize: 13 }}>ParPar (par2)</span>
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr 1fr', gap: 12 }}>
          <Field label="Taille slice" value={config.parpar.slice_size} onChange={v => update('parpar.slice_size', v)} placeholder="10M" />
          <Field label="Memoire max" value={config.parpar.memory} onChange={v => update('parpar.memory', v)} placeholder="4096M" />
          <Field label="Threads" value={String(config.parpar.threads)} onChange={v => update('parpar.threads', parseInt(v) || 16)} type="number" />
          <Field label="Redondance" value={config.parpar.redundancy} onChange={v => update('parpar.redundancy', v)} placeholder="20%" />
        </div>
      </div>

      {/* API + Output cote a cote */}
      <div style={{ display: 'flex', gap: 12 }}>
        <div className="card" style={{ flex: 1 }}>
          <div className="card-header" style={{ marginBottom: 10 }}>
            <span className="card-title" style={{ fontSize: 13 }}>API Upload</span>
            <label className="toggle">
              <input type="checkbox" checked={config.api.enabled ?? false} onChange={e => update('api.enabled', e.target.checked)} />
              <span className="toggle-slider"></span>
            </label>
          </div>
          {(config.api.enabled ?? false) ? (
            <div>
              <Field label="Cle API" value={config.api.apikey} onChange={v => update('api.apikey', v)} />
            </div>
          ) : (
            <p className="text-muted text-sm">Desactive</p>
          )}
        </div>

        <div className="card" style={{ flex: 1 }}>
          <div className="card-header" style={{ marginBottom: 10 }}>
            <span className="card-title" style={{ fontSize: 13 }}>Dossier de sortie</span>
          </div>
          <div style={{ display: 'flex', gap: 8 }}>
            <input
              className="input"
              value={config.output_dir}
              onChange={e => update('output_dir', e.target.value)}
              placeholder="Meme dossier que le fichier source"
            />
            <button className="btn btn-secondary btn-sm" onClick={async () => {
              try { const dir = await SelectDirectory(); if (dir) update('output_dir', dir) } catch {}
            }}>Changer...</button>
          </div>
          {!config.output_dir && (
            <p className="text-muted text-xs" style={{ marginTop: 6 }}>Par defaut : meme dossier que le fichier source</p>
          )}
        </div>
      </div>
    </div>
  )
}

function Field({ label, value, onChange, type = 'text', placeholder = '' }: {
  label: string; value: string; onChange: (v: string) => void; type?: string; placeholder?: string
}) {
  return (
    <div className="form-group">
      <label className="label">{label}</label>
      <input className="input" type={type} value={value} onChange={e => onChange(e.target.value)} placeholder={placeholder} />
    </div>
  )
}
