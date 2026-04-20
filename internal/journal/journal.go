package journal

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

type Entry struct {
	ID        int64  `db:"id" json:"id"`
	Level     string `db:"level" json:"level"`
	Message   string `db:"message" json:"message"`
	CreatedAt string `db:"created_at" json:"created_at"`
}

type logEntry struct {
	level   string
	message string
	at      string
}

type ListParams struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type ListResult struct {
	Entries []Entry `json:"entries"`
	Total   int     `json:"total"`
}

type DB struct {
	db   *sqlx.DB
	ch   chan logEntry
	done chan struct{}
}

func Open() (*DB, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".config", "1up")
	os.MkdirAll(dir, 0755)
	dbPath := filepath.Join(dir, "journal.db")

	db, err := sqlx.Open("sqlite", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("erreur ouverture journal db: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS journal (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			level TEXT NOT NULL DEFAULT 'info',
			message TEXT NOT NULL,
			created_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_journal_created ON journal(created_at);
	`)
	if err != nil {
		return nil, err
	}

	// Nettoyage > 24h
	cutoff := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	db.Exec("DELETE FROM journal WHERE created_at < ?", cutoff)

	j := &DB{
		db:   db,
		ch:   make(chan logEntry, 1000),
		done: make(chan struct{}),
	}
	go j.writer()
	return j, nil
}

// writer ecrit les logs en batch depuis le channel
func (j *DB) writer() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var batch []logEntry

	flush := func() {
		if len(batch) == 0 {
			return
		}
		tx, err := j.db.Begin()
		if err != nil {
			batch = nil
			return
		}
		stmt, err := tx.Prepare("INSERT INTO journal (level, message, created_at) VALUES (?, ?, ?)")
		if err != nil {
			tx.Rollback()
			batch = nil
			return
		}
		for _, e := range batch {
			stmt.Exec(e.level, e.message, e.at)
		}
		stmt.Close()
		tx.Commit()
		batch = batch[:0]
	}

	for {
		select {
		case e, ok := <-j.ch:
			if !ok {
				flush()
				close(j.done)
				return
			}
			batch = append(batch, e)
			if len(batch) >= 50 {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (j *DB) Add(level string, message string) {
	select {
	case j.ch <- logEntry{level: level, message: message, at: time.Now().Format(time.RFC3339)}:
	default: // drop si buffer plein
	}
}

func (j *DB) Info(msg string)  { j.Add("info", msg) }
func (j *DB) Error(msg string) { j.Add("error", msg) }
func (j *DB) Warn(msg string)  { j.Add("warn", msg) }

func (j *DB) List(params ListParams) (*ListResult, error) {
	cutoff := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	j.db.Exec("DELETE FROM journal WHERE created_at < ?", cutoff)

	if params.Limit <= 0 {
		params.Limit = 50
	}

	var total int
	j.db.Get(&total, "SELECT COUNT(*) FROM journal")

	var entries []Entry
	err := j.db.Select(&entries, "SELECT * FROM journal ORDER BY created_at DESC LIMIT ? OFFSET ?", params.Limit, params.Offset)
	if err != nil {
		return nil, err
	}
	if entries == nil {
		entries = []Entry{}
	}
	return &ListResult{Entries: entries, Total: total}, nil
}

func (j *DB) Clear() {
	j.db.Exec("DELETE FROM journal")
}

func (j *DB) Close() error {
	close(j.ch)
	<-j.done // attendre que le writer finisse
	return j.db.Close()
}

func (j *DB) Count() int {
	var count int
	j.db.Get(&count, "SELECT COUNT(*) FROM journal")
	return count
}

func (j *DB) Purge() int64 {
	cutoff := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	result, _ := j.db.Exec("DELETE FROM journal WHERE created_at < ?", cutoff)
	n, _ := result.RowsAffected()
	return n
}
