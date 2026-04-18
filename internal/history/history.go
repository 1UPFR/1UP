package history

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

type Entry struct {
	ID            int64  `db:"id" json:"id"`
	ReleaseName   string `db:"release_name" json:"release_name"`
	FilePath      string `db:"file_path" json:"file_path"`
	Status        string `db:"status" json:"status"`
	Error         string `db:"error" json:"error,omitempty"`
	NZBPath       string `db:"nzb_path" json:"nzb_path,omitempty"`
	Resolution    string `db:"resolution" json:"resolution,omitempty"`
	VideoCodec    string `db:"video_codec" json:"video_codec,omitempty"`
	AudioCodec    string `db:"audio_codec" json:"audio_codec,omitempty"`
	HDRFormat     string `db:"hdr_format" json:"hdr_format,omitempty"`
	FileSize      int64  `db:"file_size" json:"file_size"`
	Duration      string `db:"duration" json:"duration,omitempty"`
	AudioLangs    string `db:"audio_langs" json:"audio_langs,omitempty"`
	SubtitleLangs string `db:"subtitle_langs" json:"subtitle_langs,omitempty"`
	TMDBTitle     string `db:"tmdb_title" json:"tmdb_title,omitempty"`
	TMDBYear      string `db:"tmdb_year" json:"tmdb_year,omitempty"`
	TMDBPoster    string `db:"tmdb_poster" json:"tmdb_poster,omitempty"`
	TMDBType      string `db:"tmdb_type" json:"tmdb_type,omitempty"`
	APIResult     string `db:"api_result" json:"api_result,omitempty"`
	CreatedAt     string `db:"created_at" json:"created_at"`
}

type ListParams struct {
	Search string `json:"search"`
	Status string `json:"status"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type ListResult struct {
	Entries []Entry `json:"entries"`
	Total   int     `json:"total"`
}

type DB struct {
	db *sqlx.DB
}

func Open() (*DB, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".config", "1up")
	os.MkdirAll(dir, 0755)
	dbPath := filepath.Join(dir, "history.db")

	db, err := sqlx.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("erreur ouverture db: %w", err)
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("erreur migration db: %w", err)
	}

	return &DB{db: db}, nil
}

func migrate(db *sqlx.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			release_name TEXT NOT NULL,
			file_path TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'processing',
			error TEXT DEFAULT '',
			nzb_path TEXT DEFAULT '',
			resolution TEXT DEFAULT '',
			video_codec TEXT DEFAULT '',
			audio_codec TEXT DEFAULT '',
			hdr_format TEXT DEFAULT '',
			file_size INTEGER DEFAULT 0,
			duration TEXT DEFAULT '',
			audio_langs TEXT DEFAULT '',
			subtitle_langs TEXT DEFAULT '',
			tmdb_title TEXT DEFAULT '',
			tmdb_year TEXT DEFAULT '',
			tmdb_poster TEXT DEFAULT '',
			tmdb_type TEXT DEFAULT '',
			api_result TEXT DEFAULT '',
			created_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_history_release ON history(release_name);
		CREATE INDEX IF NOT EXISTS idx_history_status ON history(status);
		CREATE INDEX IF NOT EXISTS idx_history_created ON history(created_at);
	`)
	return err
}

func (h *DB) Add(e *Entry) (int64, error) {
	e.CreatedAt = time.Now().Format(time.RFC3339)
	result, err := h.db.NamedExec(`
		INSERT INTO history (release_name, file_path, status, error, nzb_path,
			resolution, video_codec, audio_codec, hdr_format, file_size, duration,
			audio_langs, subtitle_langs, tmdb_title, tmdb_year, tmdb_poster, tmdb_type,
			api_result, created_at)
		VALUES (:release_name, :file_path, :status, :error, :nzb_path,
			:resolution, :video_codec, :audio_codec, :hdr_format, :file_size, :duration,
			:audio_langs, :subtitle_langs, :tmdb_title, :tmdb_year, :tmdb_poster, :tmdb_type,
			:api_result, :created_at)
	`, e)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (h *DB) Update(id int64, status string, nzbPath string, apiResult string, errMsg string) error {
	_, err := h.db.Exec(`
		UPDATE history SET status = ?, nzb_path = ?, api_result = ?, error = ? WHERE id = ?
	`, status, nzbPath, apiResult, errMsg, id)
	return err
}

func (h *DB) List(params ListParams) (*ListResult, error) {
	if params.Limit <= 0 {
		params.Limit = 50
	}

	where := "1=1"
	args := []interface{}{}

	if params.Search != "" {
		where += " AND release_name LIKE ?"
		args = append(args, "%"+params.Search+"%")
	}
	if params.Status != "" {
		where += " AND status = ?"
		args = append(args, params.Status)
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := h.db.Get(&total, "SELECT COUNT(*) FROM history WHERE "+where, countArgs...)
	if err != nil {
		return nil, err
	}

	args = append(args, params.Limit, params.Offset)
	var entries []Entry
	err = h.db.Select(&entries, "SELECT * FROM history WHERE "+where+" ORDER BY created_at DESC LIMIT ? OFFSET ?", args...)
	if err != nil {
		return nil, err
	}

	if entries == nil {
		entries = []Entry{}
	}

	return &ListResult{Entries: entries, Total: total}, nil
}

func (h *DB) Delete(id int64) error {
	_, err := h.db.Exec("DELETE FROM history WHERE id = ?", id)
	return err
}

func (h *DB) Clear() error {
	_, err := h.db.Exec("DELETE FROM history")
	return err
}

func (h *DB) DB() *sqlx.DB {
	return h.db
}

func (h *DB) Close() error {
	return h.db.Close()
}

func (h *DB) Stats() (map[string]interface{}, error) {
	stats := map[string]interface{}{}

	var total int
	h.db.Get(&total, "SELECT COUNT(*) FROM history")
	stats["total"] = total

	var success int
	h.db.Get(&success, "SELECT COUNT(*) FROM history WHERE status = 'success'")
	stats["success"] = success

	var errors int
	h.db.Get(&errors, "SELECT COUNT(*) FROM history WHERE status = 'error'")
	stats["errors"] = errors

	var totalSize sql.NullInt64
	h.db.Get(&totalSize, "SELECT SUM(file_size) FROM history WHERE status = 'success'")
	if totalSize.Valid {
		stats["total_size"] = totalSize.Int64
	} else {
		stats["total_size"] = 0
	}

	return stats, nil
}
