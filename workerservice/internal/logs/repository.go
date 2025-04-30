package logs

import (
	"fmt"
	"time"

	"workerservice/config"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

type LogRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

type LogEntry struct {
	ID        int64     `db:"id"`
	Message   string    `db:"message"`
	CreatedAt time.Time `db:"created_at"`
}

func NewLogRepository(logger *zap.Logger) (*LogRepository, error) {
	db, err := sqlx.Connect("sqlite3", config.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Create logs table if it doesn't exist
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS logs (
            id INTEGER PRIMARY KEY AUTOINCREMENT,message TEXT NOT NULL,created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `)
	if err != nil {
		return nil, fmt.Errorf("failed to create logs table: %w", err)
	}

	return &LogRepository{db: db, logger: logger}, nil
}

func (r *LogRepository) StoreLog(data []byte) error {
	message := string(data)
	_, err := r.db.Exec("INSERT INTO logs (message) VALUES (?)", message)
	if err != nil {
		return fmt.Errorf("failed to insert log: %w", err)
	}

	r.logger.Info("Log stored", zap.String("message", message))
	return nil
}

func (r *LogRepository) Close() error {
	return r.db.Close()
}

func (r *LogRepository) GetRecentLogs(limit int) ([]LogEntry, error) {
	var logs []LogEntry
	err := r.db.Select(&logs, "SELECT id, message, created_at FROM logs ORDER BY created_at DESC LIMIT ?", limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	return logs, nil
}
