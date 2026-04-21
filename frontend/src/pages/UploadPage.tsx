import React, { useState, useEffect, useCallback, useRef } from 'react'
import { SelectFiles, ProcessFile, SaveMediaInfoJSON, SearchTMDB, GetTMDBDetails, CheckRelease, FindBDInfoFile } from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff, OnFileDrop, OnFileDropOff } from '../../wailsjs/runtime/runtime'
import { getMediaInfoJS, getMediaInfoJSON, type ParsedMediaInfo } from '../services/mediainfo'

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

type ItemStatus = 'pending' | 'analyzing' | 'processing' | 'done' | 'error'

interface QueueItem {
  id: string
  path: string
  name: string
  isISO: boolean
  duplicate?: boolean
  duplicateMsg?: string
  analyzed: boolean
  status: ItemStatus
  step?: string
  error?: string
  mediaInfo?: ParsedMediaInfo
  tmdbTitle?: string
  tmdbYear?: string
  tmdbPoster?: string
  tmdbType?: string
  tmdbGenres?: string[]
  tmdbOverview?: string
  tmdbRating?: number
  bdinfoPath?: string
  parparPercent: number
  nyuuPercent: number
  nyuuArticles: string
  nyuuSpeed: string
  nyuuETA: string
}

let _idCounter = 0
const ALLOWED_EXT = ['.mkv', '.mp4', '.iso']

function getExt(path: string): string {
  const dot = path.lastIndexOf('.')
  return dot >= 0 ? path.substring(dot).toLowerCase() : ''
}

function makeItem(path: string): QueueItem {
  const ext = getExt(path)
  return {
    id: String(++_idCounter),
    path,
    name: path.split(/[/\\]/).pop()?.replace(/\.[^.]+$/, '') || path,
    isISO: ext === '.iso',
    analyzed: false,
    status: 'pending',
    parparPercent: 0,
    nyuuPercent: 0,
    nyuuArticles: '',
    nyuuSpeed: '',
    nyuuETA: '',
  }
}

interface Props {
  addLog: (msg: string) => void
  logs: string[]
}

export default function UploadPage({ addLog, logs }: Props) {
  const [queue, setQueue] = useState<QueueItem[]>([])
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [dragging, setDragging] = useState(false)
  const processingRef = useRef(false)
  const queueRef = useRef(queue)
  queueRef.current = queue
  const logsEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [logs])

  const updateItem = useCallback((id: string, patch: Partial<QueueItem>) => {
    setQueue(q => q.map(item => item.id === id ? { ...item, ...patch } : item))
  }, [])

  const addFiles = useCallback((paths: string[]) => {
    const filtered = paths.filter(p => ALLOWED_EXT.includes(getExt(p)))
    if (filtered.length === 0) return
    setQueue(q => {
      const existing = new Set(q.map(i => i.path))
      const newItems = filtered.filter(p => !existing.has(p)).map(makeItem)
      if (newItems.length > 0) setSelectedId(newItems[0].id)
      return [...q, ...newItems]
    })
  }, [])

  // Drag & drop
  useEffect(() => {
    OnFileDrop((_x: any, _y: any, paths: string[]) => {
      if (paths.length > 0) addFiles(paths)
    }, true)
    return () => { OnFileDropOff() }
  }, [addFiles])

  // Events du backend
  useEffect(() => {
    EventsOn('parpar:progress', (evt: any) => {
      const qid = evt?.queueID; const data = evt?.data
      if (!qid || !data) return
      updateItem(qid, { parparPercent: data.done ? 100 : (data.percent ?? 0) })
    })
    EventsOn('nyuu:progress', (evt: any) => {
      const qid = evt?.queueID; const data = evt?.data
      if (!qid || !data) return
      updateItem(qid, {
        nyuuPercent: data.done ? 100 : (data.percent ?? 0),
        nyuuArticles: data.articles || '',
        nyuuSpeed: data.done ? '' : (data.speed || ''),
        nyuuETA: data.done ? '' : (data.eta || ''),
      })
    })
    EventsOn('status', (evt: any) => {
      const qid = evt?.queueID; const data = evt?.data
      if (!qid) return
      updateItem(qid, { step: String(data) })
      addLog(String(data))
    })
    EventsOn('upload:result', (evt: any) => addLog('Upload API: ' + (evt?.data || '')))
    return () => { EventsOff('parpar:progress'); EventsOff('nyuu:progress'); EventsOff('status'); EventsOff('upload:result') }
  }, [addLog, updateItem])

  // Analyse MediaInfo + TMDB + check API
  const analyzeItem = useCallback(async (item: QueueItem) => {
    updateItem(item.id, { status: 'analyzing', step: 'Verification...' })

    // Check API si activee
    try {
      const check = await CheckRelease(item.name)
      if (check.exists) {
        updateItem(item.id, { duplicate: true, duplicateMsg: check.Explain })
        addLog(`[${item.name}] DEJA EXISTANT: ${check.Explain}`)
      }
    } catch {}

    // MediaInfo (pas pour les ISO)
    if (!item.isISO) {
      updateItem(item.id, { step: 'Analyse MediaInfo...' })
      try {
        const mi = await getMediaInfoJS(item.path)
        updateItem(item.id, { mediaInfo: mi })
        addLog(`[${item.name}] MediaInfo OK`)
        try { const raw = await getMediaInfoJSON(item.path); await SaveMediaInfoJSON(item.path, raw) } catch {}
      } catch (e) { addLog(`[${item.name}] MediaInfo erreur: ${e}`) }
    } else {
      // ISO : chercher un fichier BDInfo compagnon
      try {
        const bdinfo = await FindBDInfoFile(item.path)
        if (bdinfo) {
          updateItem(item.id, { bdinfoPath: bdinfo })
          addLog(`[${item.name}] BDInfo trouve: ${bdinfo.split(/[/\\]/).pop()}`)
        } else {
          addLog(`[${item.name}] ISO detecte, pas de BDInfo`)
        }
      } catch {
        addLog(`[${item.name}] ISO detecte, pas de BDInfo`)
      }
    }

    // TMDB
    updateItem(item.id, { step: 'Recherche TMDB...' })
    try {
      const results = await SearchTMDB(item.name, '')
      if (results && results.length > 0) {
        const d = await GetTMDBDetails(results[0].id, results[0].mediaType)
        updateItem(item.id, { tmdbTitle: d.title, tmdbYear: d.year, tmdbPoster: d.posterPath, tmdbType: d.mediaType, tmdbGenres: d.genres, tmdbOverview: d.overview, tmdbRating: d.rating })
        addLog(`[${item.name}] TMDB: ${d.title} (${d.year})`)
      }
    } catch { addLog(`[${item.name}] TMDB: aucun resultat`) }
    updateItem(item.id, { status: 'pending', step: undefined, analyzed: true })
  }, [addLog, updateItem])

  useEffect(() => {
    const pending = queue.filter(i => i.status === 'pending' && !i.analyzed)
    for (const item of pending) analyzeItem(item)
  }, [queue.length])

  // Traitement sequentiel
  const processNext = useCallback(async () => {
    if (processingRef.current) return
    const next = queueRef.current.find(i => i.status === 'pending' && (i.mediaInfo || i.isISO))
    if (!next) return
    processingRef.current = true
    setSelectedId(next.id)
    updateItem(next.id, { status: 'processing', step: 'Demarrage...', parparPercent: 0, nyuuPercent: 0 })
    addLog(`Traitement: ${next.name}`)
    try {
      await ProcessFile(next.path, next.id)
      updateItem(next.id, { status: 'done', step: 'Termine' })
      addLog(`Termine: ${next.name}`)
    } catch (e) {
      updateItem(next.id, { status: 'error', step: undefined, error: String(e) })
      addLog(`Erreur: ${next.name} - ${e}`)
    }
    processingRef.current = false
    setTimeout(() => processNext(), 500)
  }, [addLog, updateItem])

  const handleBrowse = async () => {
    try { const paths = await SelectFiles(); if (paths && paths.length > 0) addFiles(paths) } catch (e) { addLog('Erreur: ' + e) }
  }
  const handleRemove = (id: string) => { setQueue(q => q.filter(i => i.id !== id)); if (selectedId === id) setSelectedId(null) }
  const handleClearDone = () => { setQueue(q => q.filter(i => i.status !== 'done' && i.status !== 'error')) }

  const selected = queue.find(i => i.id === selectedId)
  const pendingCount = queue.filter(i => i.status === 'pending' && (i.mediaInfo || i.isISO)).length
  const isProcessing = !!queue.find(i => i.status === 'processing')
  const doneCount = queue.filter(i => i.status === 'done').length
  const errorCount = queue.filter(i => i.status === 'error').length

  return (
    <div className="upload-page-layout">
      {/* === COLONNE GAUCHE === */}
      <div className="upload-col-left">
        {/* Drop zone compact */}
        <div
          className={`drop-zone ${dragging ? 'active' : ''}`}
          style={{ '--wails-drop-target': 'drop', padding: '20px 16px' } as React.CSSProperties}
          onDragOver={e => { e.preventDefault(); setDragging(true) }}
          onDragLeave={() => setDragging(false)}
          onClick={handleBrowse}
        >
          <div className="drop-icon" style={{ fontSize: 36, marginBottom: 6 }}>&#128194;</div>
          <p className="drop-title" style={{ fontSize: 14 }}>
            {(window as any)._1UP_WEB ? 'Cliquez pour ajouter des fichiers' : 'Glissez des fichiers ici ou cliquez pour parcourir'}
          </p>
        </div>

        {/* MediaInfo */}
        {selected?.mediaInfo && (
          <div className="card" style={{ marginTop: 12 }}>
            <div className="card-header" style={{ marginBottom: 8 }}>
              <span className="card-title" style={{ fontSize: 13 }}>MediaInfo</span>
              <span className="badge badge-success">OK</span>
            </div>
            <div className="grid-2-sep" style={{ gap: 12 }}>
              <div className="info-row"><span className="info-label">Resolution</span><span className="info-value">{selected.mediaInfo.resolution}</span></div>
              <div className="info-row"><span className="info-label">Video</span><span className="info-value">{selected.mediaInfo.videoCodec}</span></div>
              <div className="info-row"><span className="info-label">Audio</span><span className="info-value">{selected.mediaInfo.audioCodec}</span></div>
              <div className="info-row"><span className="info-label">HDR</span><span className="info-value">{selected.mediaInfo.hdrFormat || 'SDR'}</span></div>
              <div className="info-row"><span className="info-label">Taille</span><span className="info-value">{formatBytes(selected.mediaInfo.fileSize)}</span></div>
              <div className="info-row"><span className="info-label">Duree</span><span className="info-value">{selected.mediaInfo.duration}</span></div>
              <div className="info-row"><span className="info-label">Langues</span><span className="info-value">{selected.mediaInfo.audioLanguages}</span></div>
              <div className="info-row"><span className="info-label">Sous-titres</span><span className="info-value">{selected.mediaInfo.subtitleLanguages || '-'}</span></div>
            </div>
          </div>
        )}

        {/* Progression */}
        {selected && (selected.status === 'processing' || selected.status === 'done') && (
          <div className="card" style={{ marginTop: 12 }}>
            <div className="card-header" style={{ marginBottom: 8 }}>
              <span className="card-title" style={{ fontSize: 13 }}>Progression</span>
              <span className="text-secondary text-xs">{selected.step}</span>
            </div>
            <div style={{ marginBottom: 10 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 3 }}>
                <span className="text-xs">ParPar</span>
                <span className="text-xs text-secondary">{selected.parparPercent.toFixed(1)}%</span>
              </div>
              <div className="progress-bar" style={{ height: 6 }}>
                <div className={`progress-fill ${selected.parparPercent >= 100 ? 'done' : ''}`} style={{ width: `${selected.parparPercent}%` }} />
              </div>
            </div>
            <div>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 3 }}>
                <span className="text-xs">Nyuu{selected.nyuuSpeed ? ` ${selected.nyuuSpeed}` : ''}{selected.nyuuETA ? ` ETA ${selected.nyuuETA}` : ''}</span>
                <span className="text-xs text-secondary">{selected.nyuuPercent.toFixed(1)}%</span>
              </div>
              <div className="progress-bar" style={{ height: 6 }}>
                <div className={`progress-fill ${selected.nyuuPercent >= 100 ? 'done' : ''}`} style={{ width: `${selected.nyuuPercent}%` }} />
              </div>
            </div>
          </div>
        )}

        {/* Journal */}
        <div className="card" style={{ marginTop: 12, flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 }}>
          <div className="card-header" style={{ marginBottom: 8 }}>
            <span className="card-title" style={{ fontSize: 13 }}>Journal</span>
          </div>
          <div className="log-area" style={{ flex: 1, maxHeight: 'none' }}>
            {logs.length === 0 ? (
              <span className="text-muted">En attente...</span>
            ) : (
              logs.map((l, i) => <div key={i}>{l}</div>)
            )}
            <div ref={logsEndRef} />
          </div>
        </div>
      </div>

      {/* === COLONNE DROITE === */}
      <div className="upload-col-right">
        {/* TMDB */}
        {selected?.tmdbTitle ? (
          <div className="card" style={{ marginBottom: 12 }}>
            <div className="tmdb-card">
              {selected.tmdbPoster && (
                <div className="tmdb-poster">
                  <img src={selected.tmdbPoster} alt={selected.tmdbTitle} />
                </div>
              )}
              <div className="tmdb-info">
                <div className="tmdb-title" style={{ fontSize: 16 }}>
                  {selected.tmdbTitle}
                  {selected.tmdbYear && <span className="text-secondary"> ({selected.tmdbYear})</span>}
                </div>
                <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap', marginBottom: 6 }}>
                  <span className="badge badge-accent">{selected.tmdbType === 'tv' ? 'Serie' : 'Film'}</span>
                  {selected.tmdbGenres?.map(g => <span key={g} className="tag" style={{ fontSize: 11 }}>{g}</span>)}
                  {(selected.tmdbRating ?? 0) > 0 && <span className="tag tag-accent" style={{ fontSize: 11 }}>&#9733; {selected.tmdbRating!.toFixed(1)}</span>}
                </div>
                {selected.tmdbOverview && (
                  <p className="tmdb-overview" style={{ fontSize: 12 }}>{selected.tmdbOverview}</p>
                )}
              </div>
            </div>
          </div>
        ) : selected ? (
          <div className="card" style={{ marginBottom: 12, padding: 20, textAlign: 'center' }}>
            <p className="text-muted">Aucune info TMDB</p>
          </div>
        ) : null}

        {/* Queue */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 }}>
          <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
            <span className="text-sm" style={{ fontWeight: 700 }}>{queue.length} fichier{queue.length > 1 ? 's' : ''}</span>
            {pendingCount > 0 && <span className="tag" style={{ fontSize: 10, padding: '1px 6px' }}>{pendingCount} att.</span>}
            {doneCount > 0 && <span className="tag" style={{ fontSize: 10, padding: '1px 6px', color: 'var(--color-success)' }}>{doneCount} ok</span>}
            {errorCount > 0 && <span className="tag" style={{ fontSize: 10, padding: '1px 6px', color: 'var(--color-danger)' }}>{errorCount} err</span>}
          </div>
          <div style={{ display: 'flex', gap: 6 }}>
            {(doneCount > 0 || errorCount > 0) && <button className="btn btn-ghost btn-sm" onClick={handleClearDone} style={{ fontSize: 11 }}>Effacer</button>}
            {!isProcessing && pendingCount > 0 && <button className="btn btn-primary btn-sm" onClick={processNext}>&#9654; Lancer ({pendingCount})</button>}
            {isProcessing && <span className="tag tag-accent" style={{ fontSize: 11 }}><span className="spinner" style={{ width: 10, height: 10, borderWidth: 2, marginRight: 4 }} />En cours...</span>}
          </div>
        </div>

        <div className="queue-list">
          {queue.map(item => (
            <div key={item.id} className={`queue-item ${selectedId === item.id ? 'queue-item-selected' : ''}`} onClick={() => setSelectedId(item.id)}>
              <div className="queue-item-status">
                {item.status === 'done' && <span style={{ color: 'var(--color-success)', fontWeight: 700 }}>&#10003;</span>}
                {item.status === 'error' && <span style={{ color: 'var(--color-danger)', fontWeight: 700 }}>&#10007;</span>}
                {(item.status === 'processing' || item.status === 'analyzing') && <span className="spinner" style={{ width: 14, height: 14, borderWidth: 2 }} />}
                {item.status === 'pending' && <span className="text-muted">&#9675;</span>}
              </div>
              {item.tmdbPoster && <img src={item.tmdbPoster} alt="" style={{ width: 24, height: 36, borderRadius: 2, objectFit: 'cover' }} />}
              <div style={{ flex: 1, minWidth: 0 }}>
                <div className="truncate" style={{ fontWeight: 600, fontSize: 12 }}>{item.name}</div>
                {(item.status === 'processing' || item.status === 'analyzing') && item.step && <div className="text-xs text-secondary">{item.step}</div>}
                {item.duplicate && item.status === 'pending' && <div className="text-xs" style={{ color: 'var(--color-warning)' }}>Existe deja</div>}
                {item.status === 'error' && item.error && <div className="text-xs truncate" style={{ color: 'var(--color-danger)' }}>{item.error}</div>}
              </div>
              {item.status === 'processing' && (
                <div style={{ width: 60 }}>
                  <div className="progress-bar" style={{ height: 3 }}>
                    <div className="progress-fill" style={{ width: `${item.nyuuPercent > 0 ? item.nyuuPercent : item.parparPercent}%` }} />
                  </div>
                </div>
              )}
              {item.isISO && <span className="tag" style={{ fontSize: 9, padding: '0px 5px' }}>ISO</span>}
              {item.mediaInfo && item.status !== 'processing' && <span className="tag tag-accent" style={{ fontSize: 9, padding: '0px 5px' }}>{item.mediaInfo.resolution}</span>}
              {item.status === 'pending' && !isProcessing && <button className="btn btn-ghost" style={{ fontSize: 10, padding: '1px 4px' }} onClick={e => { e.stopPropagation(); handleRemove(item.id) }}>&#10005;</button>}
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
