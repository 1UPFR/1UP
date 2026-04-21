import React, { useState } from 'react'
import { SelectFileWithFilter, CheckRelease, ManualUpload } from '../../wailsjs/go/main/App'

export default function ManualPage() {
  const [releaseName, setReleaseName] = useState('')
  const [nzbPath, setNzbPath] = useState('')
  const [mediainfoPath, setMediainfoPath] = useState('')
  const [bdinfoFullPath, setBdinfoFullPath] = useState('')

  const [checking, setChecking] = useState(false)
  const [checkResult, setCheckResult] = useState<{ exists: boolean; msg: string } | null>(null)
  const [uploading, setUploading] = useState(false)
  const [uploadResult, setUploadResult] = useState<{ success: boolean; error?: string } | null>(null)

  const fileName = (path: string) => path ? path.split(/[/\\]/).pop() || path : ''

  // Extraire le nom de release du NZB
  const handlePickNZB = async () => {
    const path = await SelectFileWithFilter('Selectionner un fichier NZB', '*.nzb')
    if (!path) return
    setNzbPath(path)
    setUploadResult(null)
    setCheckResult(null)
    const name = path.split(/[/\\]/).pop()?.replace(/\.nzb$/i, '') || ''
    setReleaseName(name)

    // Auto-check
    if (name) {
      setChecking(true)
      try {
        const res = await CheckRelease(name)
        setCheckResult({ exists: res.exists, msg: res.Explain })
      } catch {}
      setChecking(false)
    }
  }

  const handlePickMediainfo = async () => {
    const path = await SelectFileWithFilter('Selectionner le JSON MediaInfo', '*.json')
    if (path) setMediainfoPath(path)
  }

  const handlePickBDInfoFull = async () => {
    const path = await SelectFileWithFilter('Selectionner BDInfo', '*.*')
    if (path) setBdinfoFullPath(path)
  }


  const handleCheck = async () => {
    if (!releaseName) return
    setChecking(true)
    setCheckResult(null)
    try {
      const res = await CheckRelease(releaseName)
      setCheckResult({ exists: res.exists, msg: res.Explain })
    } catch (e) {
      setCheckResult({ exists: false, msg: 'Erreur: ' + e })
    }
    setChecking(false)
  }

  const handleUpload = async () => {
    if (!nzbPath || !releaseName) return
    setUploading(true)
    setUploadResult(null)
    try {
      const res = await ManualUpload(releaseName, nzbPath, mediainfoPath, bdinfoFullPath)
      setUploadResult({ success: res.success, error: res.error })
    } catch (e) {
      setUploadResult({ success: false, error: String(e) })
    }
    setUploading(false)
  }

  const canUpload = nzbPath && releaseName && !uploading

  return (
    <div>
      <div className="page-header" style={{ marginBottom: 16 }}>
        <h1 className="page-title">Upload Manuel</h1>
        <p className="page-subtitle text-secondary">Envoyer un NZB existant sur l'API</p>
      </div>

      <div style={{ display: 'flex', gap: 16 }}>
        {/* Colonne gauche : fichiers */}
        <div style={{ flex: 1 }}>
          {/* NZB */}
          <div className="card" style={{ marginBottom: 12 }}>
            <div className="card-header" style={{ marginBottom: 10 }}>
              <span className="card-title" style={{ fontSize: 13 }}>Fichier NZB *</span>
            </div>
            <div style={{ display: 'flex', gap: 8 }}>
              <input className="input" value={nzbPath ? fileName(nzbPath) : ''} readOnly placeholder="Aucun fichier selectionne" />
              <button className="btn btn-secondary btn-sm" onClick={handlePickNZB}>Parcourir</button>
            </div>
          </div>

          {/* Release name */}
          <div className="card" style={{ marginBottom: 12 }}>
            <div className="card-header" style={{ marginBottom: 10 }}>
              <span className="card-title" style={{ fontSize: 13 }}>Nom de release</span>
              {checking && <span className="spinner" style={{ width: 14, height: 14, borderWidth: 2 }} />}
              {checkResult && (
                <span style={{ fontSize: 12, fontWeight: 700, color: checkResult.exists ? 'var(--color-warning)' : 'var(--color-success)' }}>
                  {checkResult.msg}
                </span>
              )}
            </div>
            <div style={{ display: 'flex', gap: 8 }}>
              <input className="input" value={releaseName} onChange={e => { setReleaseName(e.target.value); setCheckResult(null) }} placeholder="Nom de la release" />
              <button className="btn btn-ghost btn-sm" onClick={handleCheck} disabled={!releaseName || checking}>Verifier</button>
            </div>
          </div>

          {/* MediaInfo JSON */}
          <div className="card" style={{ marginBottom: 12 }}>
            <div className="card-header" style={{ marginBottom: 10 }}>
              <span className="card-title" style={{ fontSize: 13 }}>MediaInfo JSON</span>
              {mediainfoPath && <span className="badge badge-success" style={{ fontSize: 10 }}>OK</span>}
            </div>
            <div style={{ display: 'flex', gap: 8 }}>
              <input className="input" value={mediainfoPath ? fileName(mediainfoPath) : ''} readOnly placeholder="Optionnel" />
              <button className="btn btn-secondary btn-sm" onClick={handlePickMediainfo}>Parcourir</button>
              {mediainfoPath && <button className="btn btn-ghost btn-sm" onClick={() => setMediainfoPath('')}>&#10005;</button>}
            </div>
          </div>

          {/* BDInfo */}
          <div className="card">
            <div className="card-header" style={{ marginBottom: 10 }}>
              <span className="card-title" style={{ fontSize: 13 }}>BDInfo</span>
              {bdinfoFullPath && <span className="badge badge-success" style={{ fontSize: 10 }}>OK</span>}
            </div>
            <div style={{ display: 'flex', gap: 8 }}>
              <input className="input" value={bdinfoFullPath ? fileName(bdinfoFullPath) : ''} readOnly placeholder="Optionnel" />
              <button className="btn btn-secondary btn-sm" onClick={handlePickBDInfoFull}>Parcourir</button>
              {bdinfoFullPath && <button className="btn btn-ghost btn-sm" onClick={() => setBdinfoFullPath('')}>&#10005;</button>}
            </div>
          </div>
        </div>

        {/* Colonne droite : recap + action */}
        <div style={{ width: 300, flexShrink: 0 }}>
          <div className="card" style={{ marginBottom: 12 }}>
            <div className="card-header" style={{ marginBottom: 10 }}>
              <span className="card-title" style={{ fontSize: 13 }}>Recapitulatif</span>
            </div>
            <div className="info-row"><span className="info-label">NZB</span><span className="info-value">{nzbPath ? '&#10003;' : '-'}</span></div>
            <div className="info-row"><span className="info-label">MediaInfo</span><span className="info-value">{mediainfoPath ? '&#10003;' : '-'}</span></div>
            <div className="info-row"><span className="info-label">BDInfo</span><span className="info-value">{bdinfoFullPath ? '&#10003;' : '-'}</span></div>
            {checkResult && (
              <div className="info-row">
                <span className="info-label">Statut</span>
                <span className="info-value" style={{ color: checkResult.exists ? 'var(--color-warning)' : 'var(--color-success)' }}>
                  {checkResult.exists ? 'Existe deja' : 'Nouveau'}
                </span>
              </div>
            )}
          </div>

          <button
            className="btn btn-primary"
            style={{ width: '100%', justifyContent: 'center', padding: 14, fontSize: 15 }}
            onClick={handleUpload}
            disabled={!canUpload}
          >
            {uploading ? <><span className="spinner" /> Envoi...</> : 'Envoyer sur l\'API'}
          </button>

          {uploadResult && (
            <div className="card" style={{ marginTop: 12, padding: 14 }}>
              {uploadResult.success ? (
                <div style={{ color: 'var(--color-success)', fontWeight: 700, textAlign: 'center' }}>&#10003; Upload reussi</div>
              ) : (
                <div style={{ color: 'var(--color-danger)', fontSize: 13 }}>Erreur : {uploadResult.error}</div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
