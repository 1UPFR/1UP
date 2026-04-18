import React, { useEffect, useRef } from 'react'

interface Props {
  logs: string[]
  onClear: () => void
}

export default function JournalPage({ logs, onClear }: Props) {
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [logs])

  return (
    <div>
      <div className="page-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h1 className="page-title">Journal</h1>
          <p className="page-subtitle text-secondary">{logs.length} entree{logs.length > 1 ? 's' : ''}</p>
        </div>
        {logs.length > 0 && (
          <button className="btn btn-ghost btn-sm" onClick={onClear}>Effacer</button>
        )}
      </div>

      <div className="card">
        <div className="log-area" style={{ maxHeight: 'calc(100vh - 180px)' }}>
          {logs.length === 0 ? (
            <span className="text-muted">Aucune entree</span>
          ) : (
            logs.map((l, i) => <div key={i}>{l}</div>)
          )}
          <div ref={bottomRef} />
        </div>
      </div>
    </div>
  )
}
