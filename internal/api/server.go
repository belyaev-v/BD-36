package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/belyaev-v/task36/internal/storage"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	router     chi.Router
	store      storage.Store
	staticFS   http.Handler
	defaultNum int
}

func New(store storage.Store, static http.Handler) *Server {
	s := &Server{
		router:     chi.NewRouter(),
		store:      store,
		staticFS:   static,
		defaultNum: 10,
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.Get("/api/news/{limit}", s.handleNews)
	s.router.Get("/api/news", s.handleNews)
	s.router.Handle("/*", s.staticFS)
}
func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) handleNews(w http.ResponseWriter, r *http.Request) {
	limitParam := chi.URLParam(r, "limit")
	limit := s.defaultNum

	if limitParam == "" {
		if q := r.URL.Query().Get("limit"); q != "" {
			limitParam = q
		}
	}

	if limitParam != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(limitParam)); err == nil && n > 0 {
			limit = n
		}
	}

	ctx := r.Context()
	posts, err := s.store.Latest(ctx, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(posts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
