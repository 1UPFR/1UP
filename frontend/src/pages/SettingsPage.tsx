import React, { useState, useEffect, useCallback } from 'react'
import { GetConfig, SaveConfig, SelectDirectory } from '../../wailsjs/go/main/App'

interface Config {
  nyuu: { host: string; port: number; user: string; password: string; connections: number; group: string; ssl: boolean; extra_args: string }
  parpar: { slice_size: string; memory: string; threads: number; redundancy: string; extra_args: string }
  api: { url: string; apikey: string; enabled: boolean }
  output_dir: string
}

export default function SettingsPage() {
  const [config, setConfig] = useState<Config | null>(null)
  const [saved, setSaved] = useState(false)

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

  return (
    <div>
      <div className="page-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h1 className="page-title">Reglages</h1>
        {saved && <span className="badge badge-success">Sauvegarde</span>}
      </div>

      {/* Nyuu */}
      <div className="card" style={{ marginBottom: 12 }}>
        <div className="card-header" style={{ marginBottom: 10 }}>
          <span className="card-title" style={{ fontSize: 13 }}>Serveur Usenet (Nyuu)</span>
          <label className="toggle">
            <input type="checkbox" checked={config.nyuu.ssl} onChange={e => update('nyuu.ssl', e.target.checked)} />
            <span className="toggle-slider"></span>
          </label>
        </div>
        <div className="grid-3">
          <Field label="Hote" value={config.nyuu.host} onChange={v => update('nyuu.host', v)} placeholder="news.example.com" />
          <Field label="Port" value={String(config.nyuu.port)} onChange={v => update('nyuu.port', parseInt(v) || 563)} type="number" />
          <Field label="Connexions" value={String(config.nyuu.connections)} onChange={v => update('nyuu.connections', parseInt(v) || 20)} type="number" />
        </div>
        <div className="grid-3">
          <Field label="Utilisateur" value={config.nyuu.user} onChange={v => update('nyuu.user', v)} />
          <Field label="Mot de passe" value={config.nyuu.password} onChange={v => update('nyuu.password', v)} type="password" />
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
            <div className="grid-2">
              <Field label="URL API" value={config.api.url} onChange={v => update('api.url', v)} />
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
