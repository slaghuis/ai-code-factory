package sources

import (
	"database/sql"

	"github.com/slaghuis/metrics-exporter/internal/collector"
)

type Telegram struct {
	db *sql.DB
}

func NewTelegram(path string) (*Telegram, error) {
	db, err := sql.Open("sqlite3", path+"?mode=ro&_journal=WAL")
	if err != nil {
		return nil, err
	}
	return &Telegram{db: db}, nil
}

func (t *Telegram) Name() string { return "telegram" }

func (t *Telegram) Collect() error {
	var pending, orphaned int
	if err := t.db.QueryRow(
		`SELECT COUNT(*) FROM sessions WHERE status='pending'`).Scan(&pending); err != nil {
		return err
	}
	if err := t.db.QueryRow(
		`SELECT COUNT(*) FROM sessions WHERE status='pending' AND orphaned=1`).Scan(&orphaned); err != nil {
		return err
	}
	collector.TelegramPending.Set(float64(pending))
	collector.TelegramOrphaned.Set(float64(orphaned))
	return nil
}