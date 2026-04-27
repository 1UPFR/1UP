package main

import (
	"crypto/md5"
	"embed"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/1UPFR/1UP/internal/api"
	"github.com/1UPFR/1UP/internal/bdinfo"
	"github.com/1UPFR/1UP/internal/binutil"
	"github.com/1UPFR/1UP/internal/config"
	"github.com/1UPFR/1UP/internal/relparse"
	"github.com/1UPFR/1UP/internal/history"
	"github.com/1UPFR/1UP/internal/journal"
	"github.com/1UPFR/1UP/internal/nyuu"
	"github.com/1UPFR/1UP/internal/parpar"
)

//go:embed static
var staticFS embed.FS

//go:embed binaries
var embeddedBinaries embed.FS

var AppVersion = "dev"
var apiBaseURL = ""
var tmdbProxyBase = ""
var tmdbAPIKey = ""
var cfg *config.Config
var historyDB *history.DB
var journalDB *journal.DB

var authLogin string
var authPassword string

func main() {
	host := flag.String("host", "0.0.0.0", "Adresse d'ecoute")
	port := flag.Int("port", 8080, "Port d'ecoute")
	login := flag.String("login", "", "Login pour l'acces web")
	pass := flag.String("pass", "", "Mot de passe pour l'acces web")
	versionFlag := flag.Bool("version", false, "Afficher la version")
	flag.Parse()
	authLogin = *login
	authPassword = *pass

	if *versionFlag {
		fmt.Println("1UP Web", AppVersion)
		os.Exit(0)
	}

	binutil.Init(embeddedBinaries)

	var err error
	cfg, err = config.Load()
	if err != nil {
		log.Fatalf("Erreur config: %v", err)
	}
	if apiBaseURL != "" {
		api.BaseURL = apiBaseURL
	}
	historyDB, err = history.Open()
	if err != nil {
		log.Printf("Erreur historique: %v", err)
	}
	journalDB, err = journal.Open()
	if err != nil {
		log.Printf("Erreur journal: %v", err)
	}

	mux := http.NewServeMux()

	// API
	mux.HandleFunc("/api/config", handleConfig)
	mux.HandleFunc("/api/config/save", handleConfigSave)
	mux.HandleFunc("/api/version", handleVersion)
	mux.HandleFunc("/api/filesize", handleFileSize)
	mux.HandleFunc("/api/readchunk", handleReadChunk)
	mux.HandleFunc("/api/check", handleCheck)
	mux.HandleFunc("/api/process/start", handleProcessStart)
	mux.HandleFunc("/api/process/events", handleProcessEvents)
	mux.HandleFunc("/api/history", handleHistory)
	mux.HandleFunc("/api/history/stats", handleHistoryStats)
	mux.HandleFunc("/api/history/delete", handleHistoryDelete)
	mux.HandleFunc("/api/history/clear", handleHistoryClear)
	mux.HandleFunc("/api/savemediainfo", handleSaveMediaInfo)
	mux.HandleFunc("/api/browse", handleBrowse)
	mux.HandleFunc("/api/find-bdinfo", handleFindBDInfo)
	mux.HandleFunc("/api/parse-bdinfo", handleParseBDInfo)
	mux.HandleFunc("/api/journal/add", handleJournalAdd)
	mux.HandleFunc("/api/journal/list", handleJournalList)
	mux.HandleFunc("/api/journal/clear", handleJournalClear)
	mux.HandleFunc("/api/tmdb/search", handleTMDBSearch)
	mux.HandleFunc("/api/tmdb/details", handleTMDBDetails)

	// Auth
	mux.HandleFunc("/login", handleLogin)
	mux.HandleFunc("/api/login", handleAPILogin)

	// Shim Wails pour le frontend
	mux.HandleFunc("/wails-shim.js", handleShim)

	// Frontend static avec injection du shim
	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("Erreur assets: %v", err)
	}
	mux.Handle("/", shimMiddleware(http.FileServer(http.FS(staticSub))))

	addr := fmt.Sprintf("%s:%d", *host, *port)
	fmt.Printf("1UP Web %s\n", AppVersion)
	fmt.Printf("Ecoute sur http://%s\n", addr)
	var handler http.Handler = mux
	if authLogin != "" && authPassword != "" {
		fmt.Println("Authentification activee")
		handler = authMiddleware(mux)
	}
	log.Fatal(http.ListenAndServe(addr, handler))
}

// shimMiddleware injecte le shim Wails dans index.html
func shimMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			staticSub, _ := fs.Sub(staticFS, "static")
			data, err := fs.ReadFile(staticSub, "index.html")
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			html := string(data)
			html = strings.Replace(html, "<head>", `<head><script src="/wails-shim.js"></script>`, 1)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(html))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// handleShim sert le JavaScript shim qui remplace les bindings Wails
func handleShim(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Write([]byte(wailsShimJS))
}

const wailsShimJS = `
// Wails shim - remplace les bindings Wails par des appels HTTP
(function() {
  async function call(url, body) {
    const opts = body !== undefined ? { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify(body) } : {};
    const r = await fetch(url, opts);
    if (!r.ok) throw new Error(await r.text());
    const ct = r.headers.get('content-type') || '';
    if (ct.includes('json')) return r.json();
    return r.text();
  }

  window.go = { main: { App: {
    GetConfig: () => call('/api/config'),
    SaveConfig: (cfg) => call('/api/config/save', cfg),
    SelectFiles: () => _fileBrowser({mode:'file', filter:''}).then(p => p ? [p] : []),
    SelectDirectory: () => _fileBrowser({mode:'dir', filter:'dirs'}),
    SelectFileWithFilter: (title, pattern) => _fileBrowser({mode:'file', filter:'', title}),
    ProcessFile: (path, queueID) => new Promise((resolve, reject) => {
      fetch('/api/process/start', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({file_path:path, queue_id:queueID})})
        .then(r => r.json())
        .then(r => {
          const es = new EventSource('/api/process/events?id=' + encodeURIComponent(r.process_id));
          es.onerror = () => { es.close(); resolve(); };
          es.onmessage = () => {};
          ['status','parpar:progress','nyuu:progress','upload:result','done','error'].forEach(evt => {
            es.addEventListener(evt, (e) => {
              try {
                const data = JSON.parse(e.data);
                if (evt === 'error') { es.close(); reject(data.data || data); return; }
                if (evt === 'done') { es.close(); resolve(); return; }
                window._wailsEmit(evt, data);
              } catch {}
            });
          });
        }).catch(reject);
    }),
    GetFileSize: (path) => call('/api/filesize?path=' + encodeURIComponent(path)).then(r => r.size),
    ReadFileChunk: (path, offset, size) => call('/api/readchunk?path=' + encodeURIComponent(path) + '&offset=' + offset + '&size=' + size),
    SaveMediaInfoJSON: (path, json) => call('/api/savemediainfo', {path, json}),
    CheckRelease: (name) => call('/api/check?name=' + encodeURIComponent(name)),
    ManualUpload: (name, nzb, mi, bdf, bdm) => call('/api/manual-upload', {name, nzb, mi, bdf, bdm}),
    SearchTMDB: (q, t) => call('/api/tmdb/search?q=' + encodeURIComponent(q) + '&t=' + encodeURIComponent(t||'')).then(r => { console.log('[TMDB search]', r); return r; }),
    GetTMDBDetails: (id, t) => call('/api/tmdb/details?id=' + id + '&t=' + encodeURIComponent(t||'')).then(r => { console.log('[TMDB details]', r); return r; }),
    GetAppVersion: () => call('/api/version').then(r => r.version),
    CheckUpdate: () => call('/api/version').then(r => ({available: false, latest: r.version, url: ''})),
    HistoryList: (p) => call('/api/history', p),
    HistoryStats: () => call('/api/history/stats'),
    HistoryDelete: (id) => call('/api/history/delete', {id}),
    HistoryClear: () => call('/api/history/clear', {}),
    FindBDInfoFile: (path) => call('/api/find-bdinfo?path=' + encodeURIComponent(path)).then(r => r.path || ''),
    ParseBDInfo: (path) => call('/api/parse-bdinfo?path=' + encodeURIComponent(path)),
    JournalAdd: (level, msg) => call('/api/journal/add', {level, msg}),
    JournalList: (params) => call('/api/journal/list', params || {}),
    JournalClear: () => call('/api/journal/clear', {}),
    SetHistoryMediaInfo: () => {},
    SetHistoryTMDB: () => {},
  }}};

  // Runtime shim
  const listeners = {};
  window.runtime = {
    EventsOn: (name, cb) => { if (!listeners[name]) listeners[name] = []; listeners[name].push(cb); },
    EventsOnMultiple: (name, cb, max) => { if (!listeners[name]) listeners[name] = []; listeners[name].push(cb); },
    EventsOnce: (name, cb) => { if (!listeners[name]) listeners[name] = []; listeners[name].push(cb); },
    EventsOff: (name) => { delete listeners[name]; },
    EventsOffAll: () => { Object.keys(listeners).forEach(k => delete listeners[k]); },
    EventsEmit: () => {},
    OnFileDrop: () => {},
    OnFileDropOff: () => {},
    BrowserOpenURL: (url) => window.open(url, '_blank'),
    WindowReload: () => window.location.reload(),
    WindowSetTitle: () => {},
    LogPrint: console.log,
    LogTrace: console.log,
    LogDebug: console.log,
    LogInfo: console.log,
    LogWarning: console.warn,
    LogError: console.error,
    LogFatal: console.error,
  };
  window._1UP_WEB = true;
  window._wailsEmit = function(name, data) {
    (listeners[name] || []).forEach(cb => cb(data));
  };

  // File browser modal
  function _fileBrowser(opts) {
    return new Promise((resolve) => {
      const overlay = document.createElement('div');
      Object.assign(overlay.style, {
        position:'fixed', inset:0, background:'rgba(0,0,0,0.7)', zIndex:9999,
        display:'flex', alignItems:'center', justifyContent:'center',
      });
      const modal = document.createElement('div');
      Object.assign(modal.style, {
        background:'#161b26', border:'1px solid #1e293b', borderRadius:'12px',
        width:'600px', maxHeight:'80vh', display:'flex', flexDirection:'column',
        color:'#f0f2f5', fontFamily:'-apple-system,BlinkMacSystemFont,sans-serif', fontSize:'13px',
      });
      const header = document.createElement('div');
      Object.assign(header.style, {
        padding:'14px 16px', borderBottom:'1px solid #1e293b',
        display:'flex', justifyContent:'space-between', alignItems:'center',
      });
      header.innerHTML = '<b>' + (opts.title || (opts.mode==='dir'?'Selectionner un dossier':'Selectionner un fichier')) + '</b>';
      const closeBtn = document.createElement('button');
      Object.assign(closeBtn.style, {
        background:'none', border:'none', color:'#8892a4', cursor:'pointer', fontSize:'16px',
      });
      closeBtn.textContent = '\u2715';
      closeBtn.onclick = () => { document.body.removeChild(overlay); resolve(''); };
      header.appendChild(closeBtn);

      const pathBar = document.createElement('div');
      Object.assign(pathBar.style, { padding:'8px 16px', borderBottom:'1px solid #1e293b', color:'#8892a4', fontSize:'12px' });

      const list = document.createElement('div');
      Object.assign(list.style, { flex:1, overflowY:'auto', padding:'4px 0' });

      const footer = document.createElement('div');
      Object.assign(footer.style, {
        padding:'10px 16px', borderTop:'1px solid #1e293b', display:'flex', justifyContent:'flex-end', gap:'8px',
      });

      let selected = '';
      const selectBtn = document.createElement('button');
      Object.assign(selectBtn.style, {
        background:'linear-gradient(135deg,#22c55e,#06b6d4)', color:'#fff', border:'none',
        borderRadius:'6px', padding:'6px 16px', cursor:'pointer', fontWeight:'700', fontSize:'13px',
      });
      selectBtn.textContent = 'Selectionner';
      selectBtn.disabled = true;
      selectBtn.onclick = () => { document.body.removeChild(overlay); resolve(selected); };

      if (opts.mode === 'dir') {
        const selDirBtn = document.createElement('button');
        Object.assign(selDirBtn.style, {
          background:'#1a1f2e', color:'#f0f2f5', border:'1px solid #1e293b',
          borderRadius:'6px', padding:'6px 16px', cursor:'pointer', fontSize:'13px',
        });
        selDirBtn.textContent = 'Ce dossier';
        selDirBtn.onclick = () => { document.body.removeChild(overlay); resolve(currentPath); };
        footer.appendChild(selDirBtn);
      }
      footer.appendChild(selectBtn);

      modal.appendChild(header);
      modal.appendChild(pathBar);
      modal.appendChild(list);
      modal.appendChild(footer);
      overlay.appendChild(modal);
      overlay.onclick = (e) => { if(e.target===overlay){document.body.removeChild(overlay);resolve('');} };
      document.body.appendChild(overlay);

      let currentPath = '/';
      async function load(dir) {
        currentPath = dir;
        pathBar.textContent = dir;
        list.innerHTML = '<div style="padding:20px;text-align:center;color:#4b5563">Chargement...</div>';
        try {
          const r = await fetch('/api/browse?path=' + encodeURIComponent(dir) + '&filter=' + (opts.filter||''));
          const entries = await r.json();
          list.innerHTML = '';
          selected = '';
          selectBtn.disabled = true;
          entries.forEach(e => {
            const row = document.createElement('div');
            Object.assign(row.style, {
              padding:'6px 16px', cursor:'pointer', display:'flex', alignItems:'center', gap:'8px',
              borderLeft:'3px solid transparent',
            });
            row.onmouseenter = () => { row.style.background='#1c2233'; };
            row.onmouseleave = () => { if(selected!==e.path) row.style.background=''; };
            const icon = e.is_dir ? '\uD83D\uDCC1' : '\uD83D\uDCC4';
            const size = e.is_dir ? '' : (e.size > 1048576 ? (e.size/1048576).toFixed(1)+' MB' : (e.size/1024).toFixed(0)+' KB');
            row.innerHTML = '<span>'+icon+'</span><span style="flex:1">'+e.name+'</span><span style="color:#4b5563;fontSize:11px">'+size+'</span>';
            row.onclick = () => {
              if (e.is_dir) { load(e.path); }
              else {
                selected = e.path;
                selectBtn.disabled = false;
                list.querySelectorAll('div').forEach(r => { r.style.background=''; r.style.borderLeftColor='transparent'; });
                row.style.background='#1c2233';
                row.style.borderLeftColor='#22c55e';
              }
            };
            if (e.is_dir && opts.mode==='dir') {
              row.ondblclick = () => load(e.path);
              row.onclick = () => {
                selected = e.path;
                selectBtn.disabled = false;
                list.querySelectorAll('div').forEach(r => { r.style.background=''; r.style.borderLeftColor='transparent'; });
                row.style.background='#1c2233';
                row.style.borderLeftColor='#22c55e';
              };
            }
            list.appendChild(row);
          });
        } catch(err) {
          list.innerHTML = '<div style="padding:20px;text-align:center;color:#ef4444">Erreur: '+err+'</div>';
        }
      }
      load('/');
    });
  }
})();
`

// Authentification
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Page de login et assets toujours accessibles
		if r.URL.Path == "/login" || r.URL.Path == "/api/login" {
			next.ServeHTTP(w, r)
			return
		}

		// Verifier le cookie de session
		cookie, err := r.Cookie("1up_session")
		if err != nil || cookie.Value != authSessionToken() {
			if strings.HasPrefix(r.URL.Path, "/api") {
				http.Error(w, `{"error":"non autorise"}`, 401)
			} else {
				http.Redirect(w, r, "/login", 302)
			}
			return
		}

		next.ServeHTTP(w, r)
	})
}

func authSessionToken() string {
	return fmt.Sprintf("%x", md5.Sum([]byte(authLogin+":"+authPassword)))
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
<html><head><title>1UP - Login</title>
<style>
body { background: #0d1117; color: #f0f2f5; font-family: -apple-system, sans-serif; display: flex; align-items: center; justify-content: center; height: 100vh; margin: 0; }
.box { background: #161b26; border: 1px solid #1e293b; border-radius: 12px; padding: 32px; width: 320px; text-align: center; }
h1 { color: #22c55e; margin-bottom: 24px; font-size: 28px; }
input { width: 100%; padding: 10px; margin-bottom: 12px; background: #1a1f2e; border: 1px solid #1e293b; border-radius: 8px; color: #f0f2f5; font-size: 14px; box-sizing: border-box; }
input:focus { outline: none; border-color: #22c55e; }
button { width: 100%; padding: 12px; background: linear-gradient(135deg, #22c55e, #06b6d4); border: none; border-radius: 8px; color: #fff; font-size: 15px; font-weight: 700; cursor: pointer; }
.err { color: #ef4444; font-size: 13px; margin-bottom: 12px; display: none; }
</style></head><body>
<div class="box">
<h1>1UP</h1>
<div class="err" id="err">Identifiants incorrects</div>
<form onsubmit="return doLogin()">
<input type="text" id="login" placeholder="Login" autocomplete="username" autofocus />
<input type="password" id="pass" placeholder="Mot de passe" autocomplete="current-password" />
<button type="submit">Connexion</button>
</form>
</div>
<script>
function doLogin() {
  fetch('/api/login', { method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({ login: document.getElementById('login').value, password: document.getElementById('pass').value }) })
    .then(r => r.json()).then(r => { if (r.ok) window.location = '/'; else document.getElementById('err').style.display = 'block'; });
  return false;
}
</script></body></html>`)
}

func handleAPILogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "POST requis", 405)
		return
	}
	var req struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Login == authLogin && req.Password == authPassword {
		http.SetCookie(w, &http.Cookie{
			Name:     "1up_session",
			Value:    authSessionToken(),
			Path:     "/",
			MaxAge:   86400 * 30,
			HttpOnly: true,
		})
		jsonResponse(w, map[string]bool{"ok": true})
	} else {
		jsonResponse(w, map[string]bool{"ok": false})
	}
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, cfg)
}

func handleConfigSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "POST requis", 405)
		return
	}
	var newCfg config.Config
	if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
		jsonError(w, err.Error(), 400)
		return
	}
	cfg = &newCfg
	if err := cfg.Save(); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResponse(w, map[string]string{"status": "ok"})
}

func handleVersion(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]string{"version": AppVersion})
}

func handleJournalAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "POST requis", 405)
		return
	}
	var body struct {
		Level string `json:"level"`
		Msg   string `json:"msg"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, err.Error(), 400)
		return
	}
	if journalDB != nil {
		journalDB.Add(body.Level, body.Msg)
	}
	jsonResponse(w, map[string]bool{"ok": true})
}

func handleJournalList(w http.ResponseWriter, r *http.Request) {
	if journalDB == nil {
		jsonResponse(w, &journal.ListResult{Entries: []journal.Entry{}})
		return
	}
	var params journal.ListParams
	if r.Method == http.MethodPost {
		json.NewDecoder(r.Body).Decode(&params)
	}
	if params.Limit == 0 {
		params.Limit = 50
	}
	result, err := journalDB.List(params)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResponse(w, result)
}

func handleJournalClear(w http.ResponseWriter, r *http.Request) {
	if journalDB != nil {
		journalDB.Clear()
	}
	jsonResponse(w, map[string]bool{"ok": true})
}

func handleFindBDInfo(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		jsonError(w, "path requis", 400)
		return
	}
	jsonResponse(w, map[string]string{"path": bdinfo.FindFile(path)})
}

func handleParseBDInfo(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		jsonError(w, "path requis", 400)
		return
	}
	info, err := bdinfo.ParseFile(path)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResponse(w, info)
}

func handleCheck(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		jsonError(w, "name requis", 400)
		return
	}
	if !cfg.API.Enabled || cfg.API.APIKey == "" {
		jsonResponse(w, &api.CheckResult{Exists: false, Explain: "API desactivee"})
		return
	}
	result, err := api.CheckRelease(&cfg.API, name)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResponse(w, result)
}

func handleFileSize(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		jsonError(w, "path requis", 400)
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		jsonError(w, err.Error(), 404)
		return
	}
	jsonResponse(w, map[string]int64{"size": info.Size()})
}

func handleSaveMediaInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "POST requis", 405)
		return
	}
	var req struct {
		Path string `json:"path"`
		JSON string `json:"json"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, err.Error(), 400)
		return
	}
	ext := filepath.Ext(req.Path)
	name := strings.TrimSuffix(filepath.Base(req.Path), ext)
	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = filepath.Dir(req.Path)
	}
	os.MkdirAll(outputDir, 0755)
	jsonPath := filepath.Join(outputDir, name+".json")
	if err := os.WriteFile(jsonPath, []byte(req.JSON), 0644); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResponse(w, map[string]string{"path": jsonPath})
}

func handleBrowse(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("path")
	if dir == "" {
		dir = "/"
	}
	filter := r.URL.Query().Get("filter") // "files", "dirs", ou "" (tout)

	entries, err := os.ReadDir(dir)
	if err != nil {
		jsonError(w, err.Error(), 400)
		return
	}

	type Entry struct {
		Name  string `json:"name"`
		Path  string `json:"path"`
		IsDir bool   `json:"is_dir"`
		Size  int64  `json:"size"`
	}

	var result []Entry

	// Ajouter le parent
	parent := filepath.Dir(dir)
	if parent != dir {
		result = append(result, Entry{Name: "..", Path: parent, IsDir: true})
	}

	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		isDir := e.IsDir()
		if filter == "dirs" && !isDir {
			continue
		}
		var size int64
		if !isDir {
			size = info.Size()
		}
		result = append(result, Entry{
			Name:  e.Name(),
			Path:  filepath.Join(dir, e.Name()),
			IsDir: isDir,
			Size:  size,
		})
	}

	jsonResponse(w, result)
}

func handleReadChunk(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		jsonError(w, "path requis", 400)
		return
	}
	var offset int64
	var size int
	fmt.Sscanf(r.URL.Query().Get("offset"), "%d", &offset)
	fmt.Sscanf(r.URL.Query().Get("size"), "%d", &size)
	if size <= 0 {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(""))
		return
	}
	if size > 10*1024*1024 {
		size = 10 * 1024 * 1024
	}

	f, err := os.Open(path)
	if err != nil {
		jsonError(w, err.Error(), 404)
		return
	}
	defer f.Close()

	buf := make([]byte, size)
	n, err := f.ReadAt(buf, offset)
	if n == 0 {
		// EOF ou erreur : retourner une chaine vide
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(""))
		return
	}

	encoded := base64.StdEncoding.EncodeToString(buf[:n])
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(encoded))
	_ = err
}

func handleTMDBSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		jsonResponse(w, []interface{}{})
		return
	}

	type Result struct {
		ID         int     `json:"id"`
		Title      string  `json:"title"`
		Year       string  `json:"year"`
		PosterPath string  `json:"posterPath"`
		MediaType  string  `json:"mediaType"`
		Overview   string  `json:"overview"`
		Popularity float64 `json:"popularity"`
	}

	// Essayer le proxy
	if tmdbProxyBase != "" {
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(fmt.Sprintf("%s?t=search&q=%s", tmdbProxyBase, url.QueryEscape(q)))
		if err == nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			var raw struct {
				Results []struct {
					Title, EnglishTitle, OriginalTitle, Years, Poster, TmdbID, TmdbURL, ApiURL, Overview string
					NoteTmdb float64 `json:"note_tmdb"`
				} `json:"results"`
			}
			json.Unmarshal(body, &raw)
			var results []Result
			for _, r := range raw.Results {
				id := 0; fmt.Sscanf(r.TmdbID, "%d", &id)
				if id == 0 { continue }
				title := r.Title; if title == "" { title = r.EnglishTitle }; if title == "" { title = r.OriginalTitle }
				mt := "movie"; if strings.Contains(r.ApiURL, "t=tv") || strings.Contains(r.TmdbURL, "/tv/") { mt = "tv" }
				results = append(results, Result{ID: id, Title: title, Year: r.Years, PosterPath: r.Poster, MediaType: mt, Overview: r.Overview, Popularity: r.NoteTmdb})
			}
			if len(results) > 0 {
				jsonResponse(w, results)
				return
			}
		}
	}

	// Fallback API officielle
	if tmdbAPIKey != "" {
		info := relparse.Parse(q)
		if info.Title == "" { info.Title = q }
		searchType := "movie"; if info.IsTV { searchType = "tv" }

		client := &http.Client{Timeout: 10 * time.Second}
		params := url.Values{}
		params.Set("api_key", tmdbAPIKey)
		params.Set("language", "fr-FR")
		params.Set("query", info.Title)
		params.Set("page", "1")
		params.Set("include_adult", "false")
		if info.Year != "" { params.Set("year", info.Year) }

		resp, err := client.Get(fmt.Sprintf("https://api.themoviedb.org/3/search/%s?%s", searchType, params.Encode()))
		if err == nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			var raw struct {
				Results []struct {
					ID           int     `json:"id"`
					Title        string  `json:"title"`
					Name         string  `json:"name"`
					Overview     string  `json:"overview"`
					PosterPath   string  `json:"poster_path"`
					ReleaseDate  string  `json:"release_date"`
					FirstAirDate string  `json:"first_air_date"`
					VoteAverage  float64 `json:"vote_average"`
				} `json:"results"`
			}
			json.Unmarshal(body, &raw)
			var results []Result
			for _, r := range raw.Results {
				title := r.Title; if title == "" { title = r.Name }
				year := r.ReleaseDate; if year == "" { year = r.FirstAirDate }; if len(year) > 4 { year = year[:4] }
				poster := ""; if r.PosterPath != "" { poster = "https://image.tmdb.org/t/p/w200" + r.PosterPath }
				results = append(results, Result{ID: r.ID, Title: title, Year: year, PosterPath: poster, MediaType: searchType, Overview: r.Overview, Popularity: r.VoteAverage})
			}
			if results == nil { results = []Result{} }
			jsonResponse(w, results)
			return
		}
	}

	jsonResponse(w, []Result{})
}

func handleTMDBDetails(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	t := r.URL.Query().Get("t")
	if t == "" { t = "movie" }

	parseAndRespond := func(body []byte) {
		var raw struct {
			ID            int     `json:"id"`
			Title         string  `json:"title"`
			Name          string  `json:"name"`
			OriginalTitle string  `json:"original_title"`
			OriginalName  string  `json:"original_name"`
			Overview      string  `json:"overview"`
			PosterPath    string  `json:"poster_path"`
			ReleaseDate   string  `json:"release_date"`
			FirstAirDate  string  `json:"first_air_date"`
			Runtime       int     `json:"runtime"`
			VoteAverage   float64 `json:"vote_average"`
			Genres        []struct {
				Name string `json:"name"`
			} `json:"genres"`
		}
		json.Unmarshal(body, &raw)
		title := raw.Title; if title == "" { title = raw.Name }; if title == "" { title = raw.OriginalTitle }
		year := raw.ReleaseDate; if year == "" { year = raw.FirstAirDate }; if len(year) > 4 { year = year[:4] }
		var genres []string; for _, g := range raw.Genres { if g.Name != "" { genres = append(genres, g.Name) } }
		poster := ""; if raw.PosterPath != "" { poster = "https://image.tmdb.org/t/p/w200" + raw.PosterPath }
		jsonResponse(w, map[string]interface{}{
			"id": raw.ID, "title": title, "year": year, "overview": raw.Overview,
			"posterPath": poster, "mediaType": t, "genres": genres,
			"rating": raw.VoteAverage, "runtime": raw.Runtime,
		})
	}

	// Proxy
	if tmdbProxyBase != "" {
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(fmt.Sprintf("%s?t=%s&q=%s", tmdbProxyBase, t, idStr))
		if err == nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			parseAndRespond(body)
			return
		}
	}

	// Fallback officiel
	if tmdbAPIKey != "" {
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(fmt.Sprintf("https://api.themoviedb.org/3/%s/%s?api_key=%s&language=fr-FR", t, idStr, tmdbAPIKey))
		if err == nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			parseAndRespond(body)
			return
		}
	}

	jsonError(w, "aucune source TMDB disponible", 500)
}

func handleHistory(w http.ResponseWriter, r *http.Request) {
	if historyDB == nil {
		jsonResponse(w, &history.ListResult{Entries: []history.Entry{}})
		return
	}
	if r.Method == http.MethodPost {
		var params history.ListParams
		json.NewDecoder(r.Body).Decode(&params)
		result, err := historyDB.List(params)
		if err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonResponse(w, result)
		return
	}
	result, _ := historyDB.List(history.ListParams{Limit: 50})
	jsonResponse(w, result)
}

func handleHistoryStats(w http.ResponseWriter, r *http.Request) {
	if historyDB == nil {
		jsonResponse(w, map[string]interface{}{})
		return
	}
	stats, _ := historyDB.Stats()
	jsonResponse(w, stats)
}

func handleHistoryDelete(w http.ResponseWriter, r *http.Request) {
	if historyDB == nil {
		jsonResponse(w, map[string]string{"status": "ok"})
		return
	}
	var body struct{ ID int64 `json:"id"` }
	json.NewDecoder(r.Body).Decode(&body)
	historyDB.Delete(body.ID)
	jsonResponse(w, map[string]string{"status": "ok"})
}

func handleHistoryClear(w http.ResponseWriter, r *http.Request) {
	if historyDB != nil {
		historyDB.Clear()
	}
	jsonResponse(w, map[string]string{"status": "ok"})
}

// Channels d'events par processus
var (
	processChannels   = map[string]chan string{}
	processChannelsMu sync.Mutex
)

func handleProcessStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "POST requis", 405)
		return
	}

	var req struct {
		FilePath string `json:"file_path"`
		QueueID  string `json:"queue_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, err.Error(), 400)
		return
	}

	// Creer un channel pour les events
	procID := fmt.Sprintf("%s-%d", req.QueueID, time.Now().UnixNano())
	ch := make(chan string, 1000)

	processChannelsMu.Lock()
	processChannels[procID] = ch
	processChannelsMu.Unlock()

	sendEvent := func(evtType string, payload interface{}) {
		data, _ := json.Marshal(map[string]interface{}{
			"queueID": req.QueueID,
			"data":    payload,
		})
		msg := fmt.Sprintf("event: %s\ndata: %s\n\n", evtType, data)
		select {
		case ch <- msg:
		default: // drop si buffer plein
		}
	}

	// Lancer le process en goroutine
	go func() {
		defer func() {
			close(ch)
			time.Sleep(2 * time.Second)
			processChannelsMu.Lock()
			delete(processChannels, procID)
			processChannelsMu.Unlock()
		}()

		inputPath := req.FilePath
		ext := filepath.Ext(inputPath)
		releaseName := strings.TrimSuffix(filepath.Base(inputPath), ext)

		if err := relparse.ValidateTeam(releaseName); err != nil {
			sendEvent("error", err.Error())
			return
		}

		outputDir := cfg.OutputDir
		if outputDir == "" {
			outputDir = filepath.Dir(inputPath)
		}
		if !filepath.IsAbs(outputDir) {
			outputDir, _ = filepath.Abs(outputDir)
		}
		os.MkdirAll(outputDir, 0755)

		var historyID int64
		if historyDB != nil {
			historyID, _ = historyDB.Add(&history.Entry{
				ReleaseName: releaseName,
				FilePath:    inputPath,
				Status:      "processing",
			})
		}

		sendEvent("status", "Generation par2...")

		err := parpar.Run(&cfg.ParPar, inputPath, func(p parpar.Progress) {
			sendEvent("parpar:progress", p)
		})
		if err != nil {
			if historyDB != nil {
				historyDB.Update(historyID, "error", "", "", err.Error())
			}
			sendEvent("error", fmt.Sprintf("par2: %v", err))
			return
		}

		par2Pattern := filepath.Join(filepath.Dir(inputPath), releaseName+".*.par2")
		par2Files, _ := filepath.Glob(par2Pattern)
		mainPar2 := filepath.Join(filepath.Dir(inputPath), releaseName+".par2")
		allFiles := []string{inputPath}
		if _, err := os.Stat(mainPar2); err == nil {
			allFiles = append(allFiles, mainPar2)
		}
		allFiles = append(allFiles, par2Files...)

		sendEvent("status", "Post Usenet...")
		nzbPath := filepath.Join(outputDir, releaseName+".nzb")
		result, err := nyuu.Run(&cfg.Nyuu, allFiles, nzbPath, releaseName, func(p nyuu.Progress) {
			sendEvent("nyuu:progress", p)
		})
		if err != nil {
			if historyDB != nil {
				historyDB.Update(historyID, "error", "", "", err.Error())
			}
			sendEvent("error", fmt.Sprintf("nyuu: %v", err))
			return
		}

		isISO := strings.EqualFold(ext, ".iso")
		apiResultStr := ""
		if cfg.API.Enabled && cfg.API.APIKey != "" {
			if isISO {
				bdinfoPath := bdinfo.FindFile(inputPath)
				if bdinfoPath != "" {
					sendEvent("status", "Upload API (BDInfo)...")
					uploadResult, err := api.UploadISO(&cfg.API, releaseName, result.NZBPath, bdinfoPath)
					if err == nil {
						j, _ := json.Marshal(uploadResult)
						apiResultStr = string(j)
						sendEvent("upload:result", apiResultStr)
					}
				}
			} else {
				sendEvent("status", "Upload API...")
				jsonPath := filepath.Join(outputDir, releaseName+".json")
				if _, err := os.Stat(jsonPath); err == nil {
					uploadResult, err := api.Upload(&cfg.API, releaseName, result.NZBPath, jsonPath)
					if err == nil {
						j, _ := json.Marshal(uploadResult)
						apiResultStr = string(j)
						sendEvent("upload:result", apiResultStr)
					}
				}
			}
		}

		if _, err := os.Stat(mainPar2); err == nil {
			os.Remove(mainPar2)
		}
		for _, f := range par2Files {
			os.Remove(f)
		}

		if historyDB != nil {
			historyDB.Update(historyID, "success", result.NZBPath, apiResultStr, "")
		}

		sendEvent("status", "Termine")
		sendEvent("done", map[string]string{"nzb": result.NZBPath})
	}()

	jsonResponse(w, map[string]string{"process_id": procID})
}

func handleProcessEvents(w http.ResponseWriter, r *http.Request) {
	procID := r.URL.Query().Get("id")

	processChannelsMu.Lock()
	ch, ok := processChannels[procID]
	processChannelsMu.Unlock()

	if !ok {
		jsonError(w, "process non trouve", 404)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		jsonError(w, "streaming non supporte", 500)
		return
	}

	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	for msg := range ch {
		fmt.Fprint(w, msg)
		flusher.Flush()
	}
}
