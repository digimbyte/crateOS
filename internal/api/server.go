package api

import (
	"net"
	"net/http"
	"os"

	"github.com/crateos/crateos/internal/platform"
)

// Server wraps the HTTP server on a Unix socket.
type Server struct {
	srv      *http.Server
	listener net.Listener
}

// Start launches the API server on the CrateOS agent socket.
func Start() (*Server, error) {
	_ = os.Remove(platform.AgentSocket)
	if err := os.MkdirAll(platform.CratePath("runtime"), 0755); err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/status", handleStatus)
	mux.HandleFunc("/diagnostics/actors", handleActorDiagnostics)
	mux.HandleFunc("/services", handleServices)
	mux.HandleFunc("/services/enable", handleServiceEnable)
	mux.HandleFunc("/services/disable", handleServiceDisable)
	mux.HandleFunc("/services/start", handleServiceStart)
	mux.HandleFunc("/services/stop", handleServiceStop)
	mux.HandleFunc("/users", handleUsers)
	mux.HandleFunc("/users/add", handleUserAdd)
	mux.HandleFunc("/users/delete", handleUserDelete)
	mux.HandleFunc("/users/update", handleUserUpdate)
	mux.HandleFunc("/bootstrap", handleBootstrap)
	mux.HandleFunc("/uploads/ftp/complete", handleFTPUploadComplete)

	srv := &http.Server{Handler: mux}
	ln, err := net.Listen("unix", platform.AgentSocket)
	if err != nil {
		return nil, err
	}

	go srv.Serve(ln)

	return &Server{srv: srv, listener: ln}, nil
}

// Stop closes the server.
func (s *Server) Stop() {
	if s == nil {
		return
	}
	_ = s.srv.Close()
	_ = s.listener.Close()
}
