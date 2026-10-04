package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/kamil-koziol/pingo/internal/db"
	"github.com/kamil-koziol/pingo/internal/web/handler"
	"github.com/kamil-koziol/pingo/internal/web/static"
)

type Server struct {
	srv *http.Server
}

func New(addr string, q db.Querier, conn db.DBTX) *Server {
	return &Server{
		srv: &http.Server{Addr: addr, Handler: routes(q, conn)},
	}
}

func routes(q db.Querier, conn db.DBTX) http.Handler {
	dashboard := handler.NewDashboard(q, conn)

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static.FS)))
	mux.HandleFunc("GET /{$}", dashboard.Index)
	mux.HandleFunc("GET /dashboard/content", dashboard.Content)
	return mux
}

func (s *Server) Start() error {
	if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
