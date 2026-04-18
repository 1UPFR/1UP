import React, { useState, useEffect, useCallback } from 'react'
import { HistoryList, HistoryStats, HistoryDelete, HistoryClear } from '../../wailsjs/go/main/App'

function formatBytes(bytes: number): string {
  if (!bytes) return '-'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function formatDate(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  return d.toLocaleDateString('fr-FR') + ' ' + d.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })
}

interface Entry {
  id: number; release_name: string; file_path: string; status: string;
  error: string; nzb_path: string; resolution: string; video_codec: string;
  audio_codec: string; hdr_format: string; file_size: number; duration: string;
  audio_langs: string; subtitle_langs: string; tmdb_title: string; tmdb_year: string;
  tmdb_poster: string; tmdb_type: string; api_result: string; created_at: string;
}

export default function HistoryPage() {
  const [entries, setEntries] = useState<Entry[]>([])
  const [total, setTotal] = useState(0)
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [page, setPage] = useState(0)
  const [stats, setStats] = useState<any>({})
  const limit = 25

  const load = useCallback(async () => {
    try {
      const result = await HistoryList({ search, status: statusFilter, limit, offset: page * limit })
      setEntries((result.entries || []) as any)
      setTotal(result.total || 0)
    } catch (e) {
      console.error(e)
    }
  }, [search, statusFilter, page])

  const loadStats = async () => {
    try {
      const s = await HistoryStats()
      setStats(s)
    } catch {}
  }

  useEffect(() => { load(); loadStats() }, [load])

  const handleDelete = async (id: number) => {
    await HistoryDelete(id)
    load()
    loadStats()
  }

  const handleClear = async () => {
    await HistoryClear()
    load()
    loadStats()
  }

  const totalPages = Math.ceil(total / limit)
  const statusIcon = (s: string) => {
    if (s === 'success') return '\u2713'
    if (s === 'error') return '\u2717'
    return '\u2022'
  }
  const statusColor = (s: string) => {
    if (s === 'success') return 'var(--color-success)'
    if (s === 'error') return 'var(--color-danger)'
    return 'var(--color-warning)'
  }

  return (
    <div>
      <div className="page-header">
        <h1 className="page-title">Historique</h1>
        <p className="page-subtitle text-secondary">{total} traitement{total > 1 ? 's' : ''}</p>
      </div>

      {/* Stats */}
      <div style={{ display: 'flex', gap: 12, marginBottom: 16 }}>
        <div className="card" style={{ flex: 1, padding: 16, textAlign: 'center' }}>
          <div style={{ fontSize: 24, fontWeight: 800 }}>{stats.total || 0}</div>
          <div className="text-secondary text-sm">Total</div>
        </div>
        <div className="card" style={{ flex: 1, padding: 16, textAlign: 'center' }}>
          <div style={{ fontSize: 24, fontWeight: 800, color: 'var(--color-success)' }}>{stats.success || 0}</div>
          <div className="text-secondary text-sm">Succes</div>
        </div>
        <div className="card" style={{ flex: 1, padding: 16, textAlign: 'center' }}>
          <div style={{ fontSize: 24, fontWeight: 800, color: 'var(--color-danger)' }}>{stats.errors || 0}</div>
          <div className="text-secondary text-sm">Erreurs</div>
        </div>
        <div className="card" style={{ flex: 1, padding: 16, textAlign: 'center' }}>
          <div style={{ fontSize: 24, fontWeight: 800 }}>{formatBytes(stats.total_size || 0)}</div>
          <div className="text-secondary text-sm">Volume</div>
        </div>
      </div>

      {/* Filtres */}
      <div className="card" style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          <input
            className="input"
            style={{ flex: 1 }}
            placeholder="Rechercher une release..."
            value={search}
            onChange={e => { setSearch(e.target.value); setPage(0) }}
          />
          <select className="select" style={{ width: 140 }} value={statusFilter} onChange={e => { setStatusFilter(e.target.value); setPage(0) }}>
            <option value="">Tous</option>
            <option value="success">Succes</option>
            <option value="error">Erreurs</option>
            <option value="processing">En cours</option>
          </select>
          {total > 0 && (
            <button className="btn btn-ghost btn-sm" onClick={handleClear}>Tout effacer</button>
          )}
        </div>
      </div>

      {/* Liste */}
      {entries.length === 0 ? (
        <div className="card" style={{ textAlign: 'center', padding: 40 }}>
          <p className="text-muted">Aucun traitement enregistre</p>
        </div>
      ) : (
        <div>
          {entries.map(e => (
            <div key={e.id} className="card" style={{ marginBottom: 8, padding: 14 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                {e.tmdb_poster && (
                  <img src={e.tmdb_poster} alt="" style={{ width: 40, height: 60, borderRadius: 4, objectFit: 'cover' }} />
                )}
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <span style={{ color: statusColor(e.status), fontWeight: 700, fontSize: 16 }}>{statusIcon(e.status)}</span>
                    <span style={{ fontWeight: 700, fontSize: 14 }} className="truncate">{e.release_name}</span>
                  </div>
                  <div style={{ display: 'flex', gap: 6, marginTop: 4, flexWrap: 'wrap' }}>
                    {e.tmdb_title && <span className="tag" style={{ fontSize: 11 }}>{e.tmdb_title}{e.tmdb_year ? ` (${e.tmdb_year})` : ''}</span>}
                    {e.resolution && <span className="tag tag-accent" style={{ fontSize: 11 }}>{e.resolution}</span>}
                    {e.video_codec && <span className="tag" style={{ fontSize: 11 }}>{e.video_codec}</span>}
                    {e.audio_codec && <span className="tag" style={{ fontSize: 11 }}>{e.audio_codec}</span>}
                    {e.file_size > 0 && <span className="tag" style={{ fontSize: 11 }}>{formatBytes(e.file_size)}</span>}
                  </div>
                  {e.error && <div className="text-sm" style={{ color: 'var(--color-danger)', marginTop: 4 }}>{e.error}</div>}
                </div>
                <div style={{ textAlign: 'right', flexShrink: 0 }}>
                  <div className="text-xs text-muted">{formatDate(e.created_at)}</div>
                  <button className="btn btn-ghost btn-sm" style={{ marginTop: 4, fontSize: 11 }} onClick={() => handleDelete(e.id)}>Supprimer</button>
                </div>
              </div>
            </div>
          ))}

          {/* Pagination */}
          {totalPages > 1 && (
            <div style={{ display: 'flex', justifyContent: 'center', gap: 8, marginTop: 16 }}>
              <button className="btn btn-ghost btn-sm" disabled={page === 0} onClick={() => setPage(p => p - 1)}>Precedent</button>
              <span className="text-secondary text-sm" style={{ lineHeight: '30px' }}>{page + 1} / {totalPages}</span>
              <button className="btn btn-ghost btn-sm" disabled={page >= totalPages - 1} onClick={() => setPage(p => p + 1)}>Suivant</button>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
