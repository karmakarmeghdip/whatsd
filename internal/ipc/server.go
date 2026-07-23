package ipc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"

	"whatsd/internal/types"
	"whatsd/internal/whatsapp"
)

// Server handles IPC incoming Unix domain socket connections.
type Server struct {
	socketPath string
	waClient   *whatsapp.Client
	listener   net.Listener
	mu         sync.Mutex
	clients    map[net.Conn]struct{}
	stopCh     chan struct{}
	router     *Router
}

// NewServer creates a new IPC server instance.
func NewServer(socketPath string, waClient *whatsapp.Client) *Server {
	return &Server{
		socketPath: socketPath,
		waClient:   waClient,
		clients:    make(map[net.Conn]struct{}),
		stopCh:     make(chan struct{}),
		router:     NewRouter(),
	}
}

// Start begins listening on the Unix domain socket.
func (s *Server) Start(ctx context.Context) error {
	// Remove stale socket file if it exists
	if err := os.Remove(s.socketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove old socket file %s: %w", s.socketPath, err)
	}

	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on unix socket %s: %w", s.socketPath, err)
	}
	s.listener = listener

	// Ensure restricted permissions on socket file
	if err := os.Chmod(s.socketPath, 0600); err != nil {
		slog.Warn("failed to set restrictive permissions on socket file", "path", s.socketPath, "err", err)
	}

	slog.Info("IPC server listening", "socket", s.socketPath)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-s.stopCh:
					return
				default:
					slog.Error("failed to accept IPC connection", "err", err)
					continue
				}
			}
			s.registerClient(conn)
			go s.handleConnection(ctx, conn)
		}
	}()

	return nil
}

// Broadcast sends an event notification to all connected IPC clients.
func (s *Server) Broadcast(evt types.EventNotification) {
	data, err := json.Marshal(evt)
	if err != nil {
		slog.Error("failed to marshal event notification", "err", err)
		return
	}
	data = append(data, '\n')

	s.mu.Lock()
	defer s.mu.Unlock()

	for conn := range s.clients {
		_, err := conn.Write(data)
		if err != nil {
			slog.Debug("failed to write event to IPC client, closing connection", "err", err)
			_ = conn.Close()
			delete(s.clients, conn)
		}
	}
}

// Stop closes all connections and the socket listener.
func (s *Server) Stop() {
	s.mu.Lock()
	select {
	case <-s.stopCh:
		s.mu.Unlock()
		return
	default:
		close(s.stopCh)
	}

	if s.listener != nil {
		_ = s.listener.Close()
	}
	for conn := range s.clients {
		_ = conn.Close()
	}
	s.clients = make(map[net.Conn]struct{})
	s.mu.Unlock()

	_ = os.Remove(s.socketPath)
	slog.Info("IPC server stopped")
}

func (s *Server) registerClient(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[conn] = struct{}{}
}

func (s *Server) unregisterClient(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, conn)
	_ = conn.Close()
}

func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	defer s.unregisterClient(conn)
	scanner := bufio.NewScanner(conn)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req types.Request
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendError(conn, "", fmt.Sprintf("invalid request payload: %v", err))
			continue
		}

		s.handleRequest(ctx, conn, req)
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		slog.Debug("IPC scanner error", "err", err)
	}
}

func (s *Server) handleRequest(ctx context.Context, conn net.Conn, req types.Request) {
	s.router.Dispatch(ctx, s, conn, req)
}

func (s *Server) sendResult(conn net.Conn, id string, result any) {
	resp := types.Response{
		ID:     id,
		Result: result,
	}
	s.sendResponse(conn, resp)
}

func (s *Server) sendError(conn net.Conn, id string, errMsg string) {
	resp := types.Response{
		ID:    id,
		Error: errMsg,
	}
	s.sendResponse(conn, resp)
}

func (s *Server) sendResponse(conn net.Conn, resp types.Response) {
	data, err := json.Marshal(resp)
	if err != nil {
		slog.Error("failed to marshal response", "err", err)
		return
	}
	data = append(data, '\n')
	_, _ = conn.Write(data)
}
