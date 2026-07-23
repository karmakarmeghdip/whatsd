package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"whatsd/internal/config"
	"whatsd/internal/history"
	"whatsd/internal/ipc"
	"whatsd/internal/store"
	"whatsd/internal/types"
	"whatsd/internal/whatsapp"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("starting whatsd WhatsApp daemon")

	cfg := config.LoadConfig()
	if err := cfg.EnsureDirectories(); err != nil {
		slog.Error("failed to create runtime directories", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	container, deviceStore, db, err := store.InitStore(ctx, cfg.DBPath, nil)
	if err != nil {
		slog.Error("failed to initialize SQLite store", "err", err)
		os.Exit(1)
	}
	defer func() {
		_ = container.Close()
	}()

	historyStore, err := history.NewStore(db)
	if err != nil {
		slog.Error("failed to initialize history store", "err", err)
		os.Exit(1)
	}

	var ipcServer *ipc.Server

	waClient, err := whatsapp.NewClient(deviceStore, historyStore, func(evt types.EventNotification) {
		if ipcServer != nil {
			ipcServer.Broadcast(evt)
		}
	})
	if err != nil {
		slog.Error("failed to initialize WhatsApp client", "err", err)
		os.Exit(1)
	}

	ipcServer = ipc.NewServer(cfg.SocketPath, waClient)
	if err := ipcServer.Start(ctx); err != nil {
		slog.Error("failed to start IPC server", "err", err)
		os.Exit(1)
	}
	defer ipcServer.Stop()

	// If device already has logged-in credentials, connect automatically
	if waClient.IsLoggedIn() {
		slog.Info("existing session found, connecting to WhatsApp...")
		if err := waClient.Connect(ctx); err != nil {
			slog.Warn("auto-connect failed", "err", err)
		}
	} else {
		slog.Info("no existing session found. Waiting for IPC pair command...")
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	slog.Info("shutting down whatsd daemon...")
	waClient.Disconnect()
}
