package main

import (
	"github.com/xiaoqianran/mygpt-cf-tunnel-mcp/internal/agent"
	"github.com/xiaoqianran/mygpt-cf-tunnel-mcp/internal/config"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "error", err)
		os.Exit(1)
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	srv, err := agent.New(cfg, log)
	if err != nil {
		log.Error("registry", "error", err)
		os.Exit(1)
	}
	defer srv.Close()
	log.Info("starting", "addr", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, srv.Handler()); err != nil {
		log.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
