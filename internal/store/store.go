package store

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	_ "modernc.org/sqlite"

	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// InitStore initializes the SQLite database container for whatsmeow sessions.
func InitStore(ctx context.Context, dbPath string, logger waLog.Logger) (*sqlstore.Container, *store.Device, error) {
	if logger == nil {
		logger = waLog.Stdout("Database", "INFO", true)
	}

	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_foreign_keys=on", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open sqlite database at %s: %w", dbPath, err)
	}

	db.SetMaxOpenConns(1)
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON;"); err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("failed to enable SQLite foreign keys: %w", err)
	}

	container := sqlstore.NewWithDB(db, "sqlite", logger)
	if err := container.Upgrade(ctx); err != nil {
		_ = container.Close()
		return nil, nil, fmt.Errorf("failed to upgrade database: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		_ = container.Close()
		return nil, nil, fmt.Errorf("failed to get first device from store: %w", err)
	}

	slog.Info("initialized whatsmeow database store", "path", dbPath)
	return container, deviceStore, nil
}
