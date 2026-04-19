import { GetFileSize, ReadFileChunk } from '../../wailsjs/go/main/App'

export interface ParsedMediaInfo {
  resolution: string
  videoCodec: string
  audioCodec: string
  audioLanguages: string
  subtitleLanguages: string
  hdrFormat: string
  duration: string
  fileSize: number
  width: number
  height: number
  bitrate: number
  frameRate: number
}

declare global {
  interface Window {
    MediaInfoLib: any
  }
}

// Charger le module MediaInfoWasm
let modulePromise: Promise<any> | null = null

function loadModule(): Promise<any> {
  if (modulePromise) return modulePromise
  modulePromise = new Promise((resolve) => {
    const script = document.createElement('script')
    script.src = 'MediaInfoWasm.js'
    script.onload = () => {
      const factory = (window as any).MediaInfoLib
      if (typeof factory === 'function') {
        const mod = factory({
          locateFile: (name: string) => name,
          postRun: [] as any[],
        })
        if (mod instanceof Promise) {
          mod.then(resolve)
        } else {
          // postRun callback
          const orig = factory
          const modWithPost = orig({
            locateFile: (name: string) => name,
            postRun: [() => {}],
          })
          if (modWithPost instanceof Promise) {
            modWithPost.then(resolve)
          } else {
            resolve(modWithPost)
          }
        }
      }
    }
    document.body.appendChild(script)
  })
  return modulePromise
}

// Analyser un fichier via Wails (lecture par chunks)
async function analyzeFileNative(filePath: string): Promise<{ parsed: any; json: string }> {
  const mod = await loadModule()
  const MI = new mod.MediaInfo()
  const fileSize = await GetFileSize(filePath)
  const CHUNK_SIZE = 1024 * 1024

  // Open
  MI.Open_Buffer_Init(fileSize, 0)

  let offset = 0
  while (offset < fileSize) {
    const size = Math.min(CHUNK_SIZE, fileSize - offset)
    const b64 = await ReadFileChunk(filePath, offset, size)
    const bin = atob(b64)
    const arr = new Uint8Array(bin.length)
    for (let i = 0; i < bin.length; i++) arr[i] = bin.charCodeAt(i)

    // Feed data
    const result = MI.Open_Buffer_Continue(arr)
    const seekTo = MI.Open_Buffer_Continue_Goto_Get()

    if (seekTo !== -1) {
      offset = Number(seekTo)
      MI.Open_Buffer_Init(fileSize, offset)
    } else {
      offset += size
    }

    if (result & 0x08) break // Finished
  }

  MI.Open_Buffer_Finalize()

  // Get JSON full
  MI.Option("Inform", "JSON")
  MI.Option("Complete", "1")
  const json = MI.Inform()

  MI.Close()
  MI.delete()

  // Injecter le nom du fichier (non disponible en mode buffer)
  const fileName = filePath.split(/[/\\]/).pop() || ''
  const baseName = fileName.replace(/\.[^.]+$/, '')
  const ext = fileName.includes('.') ? fileName.substring(fileName.lastIndexOf('.') + 1) : ''

  // Injection dans le JSON string natif pour garder le formatage
  let fixedJson = json.replace('"@ref":""', '"@ref":"' + fileName + '"')

  // Injecter CompleteName apres le premier @type General
  const insertAfter = '"StreamKindID":"0",'
  const insertFields = '\n"CompleteName":"' + fileName + '",\n"FileNameExtension":"' + fileName + '",\n"FileName":"' + baseName + '",\n"FileExtension":"' + ext + '",'
  const idx = fixedJson.indexOf(insertAfter)
  if (idx !== -1) {
    fixedJson = fixedJson.slice(0, idx + insertAfter.length) + insertFields + fixedJson.slice(idx + insertAfter.length)
  }

  return { parsed: JSON.parse(fixedJson), json: fixedJson }
}

const langNames: Record<string, string> = {
  fr: 'Francais', fre: 'Francais', fra: 'Francais',
  en: 'Anglais', eng: 'Anglais',
  de: 'Allemand', ger: 'Allemand', deu: 'Allemand',
  es: 'Espagnol', spa: 'Espagnol',
  it: 'Italien', ita: 'Italien',
  ja: 'Japonais', jpn: 'Japonais',
}

function normalizeLang(lang: string): string {
  return langNames[lang?.toLowerCase()] ?? lang ?? ''
}

function detectResolution(w: number, h: number): string {
  if (h >= 1960 || w >= 3790) return '2160p'
  if (h >= 880 || w >= 1870) return '1080p'
  if (h >= 520 || w >= 1230) return '720p'
  return 'SD'
}

function normalizeVideoCodec(format: string): string {
  const f = format?.toLowerCase() ?? ''
  if (f.includes('hevc') || f === 'h.265') return 'H.265'
  if (f.includes('avc') || f === 'h.264') return 'H.264'
  if (f === 'av1') return 'AV1'
  return format ?? ''
}

function normalizeAudioCodec(format: string, profile: string, commercial: string): string {
  const c = (commercial || '').toLowerCase()
  if (c.includes('atmos')) return 'Atmos'
  if (c.includes('truehd')) return 'TrueHD'
  if (c.includes('dts-hd master')) return 'DTS-HD MA'
  if (c.includes('dts-hd')) return 'DTS-HD'
  if (c.includes('dolby digital plus')) return 'EAC3'
  if (c.includes('dolby digital')) return 'AC3'
  const f = (format || '').toUpperCase()
  if (f === 'DTS') {
    if (profile?.includes('MA')) return 'DTS-HD MA'
    return 'DTS'
  }
  if (f === 'AC-3') return 'AC3'
  if (f === 'E-AC-3') return 'EAC3'
  if (f === 'AAC') return 'AAC'
  if (f === 'FLAC') return 'FLAC'
  return f
}

function detectHDR(video: any): string {
  const hdr = (video.HDR_Format || '').toLowerCase()
  const compat = (video.HDR_Format_Compatibility || '').toLowerCase()
  const transfer = (video.transfer_characteristics || '').toLowerCase()
  if (hdr.includes('dolby vision') && (hdr.includes('hdr10') || compat.includes('hdr10'))) return 'HDR DV'
  if (hdr.includes('dolby vision')) return 'DV'
  if (hdr.includes('hdr10+')) return 'HDR10+'
  if (hdr.includes('smpte st 2086') || compat.includes('hdr10')) return 'HDR10'
  if (transfer.includes('pq') || transfer.includes('smpte st 2084')) return 'HDR10'
  if (transfer.includes('hlg')) return 'HLG'
  return ''
}

export async function getMediaInfoJS(filePath: string): Promise<ParsedMediaInfo> {
  const { parsed } = await analyzeFileNative(filePath)
  const tracks: any[] = parsed?.media?.track ?? []

  const general = tracks.find((t: any) => t['@type'] === 'General') ?? {}
  const video = tracks.find((t: any) => t['@type'] === 'Video') ?? {}
  const audioTracks = tracks.filter((t: any) => t['@type'] === 'Audio')
  const textTracks = tracks.filter((t: any) => t['@type'] === 'Text')

  const w = parseInt(video.Width ?? '0')
  const h = parseInt(video.Height ?? '0')

  let duration = ''
  const dur = parseFloat(general.Duration ?? '0')
  if (dur > 0) {
    const hh = Math.floor(dur / 3600)
    const mm = Math.floor((dur % 3600) / 60)
    duration = `${hh}h ${String(mm).padStart(2, '0')}min`
  }

  const langs = new Set<string>()
  for (const t of audioTracks) {
    const lang = normalizeLang(t.Language ?? '')
    if (lang) langs.add(lang)
  }

  const subs = new Set<string>()
  for (const t of textTracks) {
    const lang = normalizeLang(t.Language ?? '') || (t.Title ?? '')
    if (!lang) continue
    const forced = t.Forced === 'Yes' ? ' (force)' : ''
    subs.add(lang + forced)
  }

  let audioCodec = ''
  if (audioTracks.length > 0) {
    const t = audioTracks[0]
    audioCodec = normalizeAudioCodec(t.Format ?? '', t.Format_Profile ?? '', t.Format_Commercial_IfAny ?? '')
  }

  return {
    resolution: detectResolution(w, h),
    videoCodec: normalizeVideoCodec(video.Format ?? ''),
    audioCodec,
    audioLanguages: Array.from(langs).join(', '),
    subtitleLanguages: Array.from(subs).join(', '),
    hdrFormat: detectHDR(video),
    duration,
    fileSize: parseInt(general.FileSize ?? '0'),
    width: w,
    height: h,
    bitrate: parseInt(general.OverallBitRate ?? '0'),
    frameRate: Math.round(parseFloat(video.FrameRate ?? '0') * 100) / 100,
  }
}

export async function getMediaInfoJSON(filePath: string): Promise<string> {
  const { json } = await analyzeFileNative(filePath)
  return json
}
