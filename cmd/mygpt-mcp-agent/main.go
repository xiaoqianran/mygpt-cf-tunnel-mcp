package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/xiaoqianran/mygpt-cf-tunnel-mcp/internal/agent"
	"github.com/xiaoqianran/mygpt-cf-tunnel-mcp/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "error", err)
		os.Exit(1)
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	srv := agent.New(cfg, log)
	log.Info("starting", "addr", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, srv.Handler()); err != nil {
		log.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
