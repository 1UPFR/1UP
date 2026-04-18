package web

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
)

type Server struct {
	assets embed.FS
	port   int
}

func NewServer(assets embed.FS, port int) *Server {
	return &Server{assets: assets, port: port}
}

func (s *Server) Start() error {
	frontend, err := fs.Sub(s.assets, "frontend/dist")
	if err != nil {
		return fmt.Errorf("erreur assets frontend: %w", err)
	}

	http.Handle("/", http.FileServer(http.FS(frontend)))

	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("1UP Web - Écoute sur http://0.0.0.0%s\n", addr)
	return http.ListenAndServe(addr, nil)
}
