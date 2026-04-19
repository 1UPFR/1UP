import React, { useState, useEffect, useRef, useCallback } from 'react'
import { JournalList, JournalClear } from '../../wailsjs/go/main/App'

interface JEntry {
  id: number
  level: string
  message: string
  created_at: string
}

function formatTime(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

function formatDate(iso: string): string {
  const d = new Date(iso)
  const today = new Date()
  if (d.toDateString() === today.toDateString()) return "Aujourd'hui"
  const yesterday = new Date(today); yesterday.setDate(yesterday.getDate() - 1)
  if (d.toDateString() === yesterday.toDateString()) return 'Hier'
  return d.toLocaleDateString('fr-FR', { day: '2-digit', month: '2-digit' })
}

function groupByDate(entries: JEntry[]): Map<string, JEntry[]> {
  const groups = new Map<string, JEntry[]>()
  for (const e of entries) {
    const key = formatDate(e.created_at)
    if (!groups.has(key)) groups.set(key, [])
    groups.get(key)!.push(e)
  }
  return groups
}

const levelStyle: Record<string, { color: string; label: string }> = {
  info:  { color: 'var(--color-accent)', label: 'INFO' },
  error: { color: 'var(--color-danger)', label: 'ERREUR' },
  warn:  { color: 'var(--color-warning)', label: 'WARN' },
}

export default function JournalPage({ visible }: { visible?: boolean }) {
  const [entries, setEntries] = useState<JEntry[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(0)
  const [perPage, setPerPage] = useState(30)
  const contentRef = useRef<HTMLDivElement>(null)

  const calcPageSize = useCallback(() => {
    const el = contentRef.current
    if (!el) return
    // hauteur dispo - header(80) - pagination(50) - marges
    const available = el.clientHeight - 130
    const rowHeight = 36 // hauteur d'une ligne de journal
    const count = Math.max(5, Math.floor(available / rowHeight))
    setPerPage(count)
  }, [])

  useEffect(() => {
    calcPageSize()
    window.addEventListener('resize', calcPageSize)
    return () => window.removeEventListener('resize', calcPageSize)
  }, [calcPageSize])

  const load = async () => {
    try {
      const result = await JournalList({ limit: perPage, offset: page * perPage })
      setEntries((result.entries || []) as any)
      setTotal(result.total || 0)
    } catch {}
  }

  useEffect(() => { load() }, [page, perPage])
  useEffect(() => { if (visible) { setPage(0); calcPageSize(); load() } }, [visible])

  const handleClear = async () => {
    await JournalClear()
    setEntries([])
    setTotal(0)
    setPage(0)
  }

  const groups = groupByDate(entries)
  const totalPages = Math.ceil(total / perPage)

  return (
    <div ref={contentRef} style={{ height: '100%' }}>
      <div className="page-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <div>
          <h1 className="page-title">Journal</h1>
          <p className="page-subtitle text-secondary">24 dernieres heures — {total} entree{total > 1 ? 's' : ''}</p>
        </div>
        {entries.length > 0 && (
          <button className="btn btn-ghost btn-sm" onClick={handleClear}>Effacer</button>
        )}
      </div>

      {entries.length === 0 ? (
        <div className="card" style={{ textAlign: 'center', padding: 40 }}>
          <p className="text-muted">Aucune activite dans les dernieres 24 heures</p>
        </div>
      ) : (
        <>
          {Array.from(groups.entries()).map(([date, items]) => (
            <div key={date} style={{ marginBottom: 16 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 8 }}>
                <span className="text-muted text-xs" style={{ fontWeight: 700, letterSpacing: 1 }}>{date}</span>
                <div style={{ flex: 1, height: 1, background: 'var(--border-color)' }} />
                <span className="text-muted text-xs">{items.length}</span>
              </div>

              <div className="card" style={{ padding: 0, overflow: 'hidden' }}>
                {items.map((e, i) => {
                  const ls = levelStyle[e.level] || levelStyle.info
                  return (
                    <div
                      key={e.id}
                      style={{
                        display: 'flex', alignItems: 'flex-start', gap: 10,
                        padding: '8px 14px',
                        borderBottom: i < items.length - 1 ? '1px solid var(--border-color)' : 'none',
                        borderLeft: `3px solid ${ls.color}`,
                      }}
                    >
                      <span className="font-mono text-xs text-muted" style={{ flexShrink: 0, marginTop: 1 }}>
                        {formatTime(e.created_at)}
                      </span>
                      <span style={{ fontSize: 10, fontWeight: 700, color: ls.color, flexShrink: 0, marginTop: 2, width: 50 }}>
                        {ls.label}
                      </span>
                      <span style={{ fontSize: 13, wordBreak: 'break-all', color: e.level === 'error' ? 'var(--color-danger)' : 'var(--text-primary)' }}>
                        {e.message}
                      </span>
                    </div>
                  )
                })}
              </div>
            </div>
          ))}

          {totalPages > 1 && (
            <div style={{ display: 'flex', justifyContent: 'center', gap: 8, marginTop: 16 }}>
              <button className="btn btn-ghost btn-sm" disabled={page === 0} onClick={() => setPage(p => p - 1)}>Precedent</button>
              <span className="text-secondary text-sm" style={{ lineHeight: '30px' }}>{page + 1} / {totalPages}</span>
              <button className="btn btn-ghost btn-sm" disabled={page >= totalPages - 1} onClick={() => setPage(p => p + 1)}>Suivant</button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
