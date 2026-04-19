import React, { useState, useEffect, useRef } from 'react'
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
  const bottomRef = useRef<HTMLDivElement>(null)

  const load = async () => {
    try {
      const list = await JournalList()
      setEntries((list || []) as any)
    } catch {}
  }

  useEffect(() => { load() }, [])
  useEffect(() => { if (visible) load() }, [visible])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [entries])

  const handleClear = async () => {
    await JournalClear()
    setEntries([])
  }

  const groups = groupByDate(entries)

  return (
    <div>
      <div className="page-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <div>
          <h1 className="page-title">Journal</h1>
          <p className="page-subtitle text-secondary">Dernières 24 heures — {entries.length} entree{entries.length > 1 ? 's' : ''}</p>
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
        <div style={{ maxHeight: 'calc(100vh - 140px)', overflowY: 'auto' }}>
          {Array.from(groups.entries()).map(([date, items]) => (
            <div key={date} style={{ marginBottom: 16 }}>
              <div style={{
                display: 'flex', alignItems: 'center', gap: 12, marginBottom: 8
              }}>
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
          <div ref={bottomRef} />
        </div>
      )}
    </div>
  )
}
