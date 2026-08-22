package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/xiaoqianran/mygpt-cf-tunnel-mcp/internal/config"
)

func usage() {
	fmt.Fprintln(os.Stderr, `Usage:
  mygpt-mcpctl list
  mygpt-mcpctl validate
  mygpt-mcpctl add-http NAME URL [DESCRIPTION]
  mygpt-mcpctl add-stdio NAME COMMAND [ARG...]
  mygpt-mcpctl remove NAME
  mygpt-mcpctl enable NAME
  mygpt-mcpctl disable NAME

Environment:
  MCP_REGISTRY  registry path (default /etc/mygpt-mcp/servers.json)`)
}

func main() {
	path := os.Getenv("MCP_REGISTRY")
	if path == "" {
		path = "/etc/mygpt-mcp/servers.json"
	}
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	reg, err := loadOrEmpty(path)
	if err != nil {
		fatal(err)
	}
	switch os.Args[1] {
	case "list":
		names := make([]string, 0, len(reg.Servers))
		for n := range reg.Servers {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			srv := reg.Servers[n]
			target := srv.URL
			if srv.TransportName() == "stdio" {
				target = srv.Command
				for _, a := range srv.Args {
					target += " " + a
				}
			}
			state := "enabled"
			if !srv.IsEnabled() {
				state = "disabled"
			}
			fmt.Printf("%s\t%s\t%s\t%s\n", n, srv.TransportName(), state, target)
		}
	case "validate":
		for n, s := range reg.Servers {
			if err := s.Validate(n); err != nil {
				fatal(err)
			}
		}
		fmt.Printf("OK: %d MCP servers\n", len(reg.Servers))
	case "add-http":
		if len(os.Args) < 4 {
			usage()
			os.Exit(2)
		}
		d := ""
		if len(os.Args) > 4 {
			d = os.Args[4]
		}
		reg.Servers[os.Args[2]] = config.Server{Transport: "streamable_http", URL: os.Args[3], Description: d}
		mustSave(path, reg)
		fmt.Println("added", os.Args[2])
	case "add-stdio":
		if len(os.Args) < 4 {
			usage()
			os.Exit(2)
		}
		reg.Servers[os.Args[2]] = config.Server{Transport: "stdio", Command: os.Args[3], Args: append([]string(nil), os.Args[4:]...)}
		mustSave(path, reg)
		fmt.Println("added", os.Args[2])
	case "remove":
		if len(os.Args) != 3 {
			usage()
			os.Exit(2)
		}
		if _, ok := reg.Servers[os.Args[2]]; !ok {
			fatal(fmt.Errorf("unknown server %q", os.Args[2]))
		}
		delete(reg.Servers, os.Args[2])
		mustSave(path, reg)
		fmt.Println("removed", os.Args[2])
	case "enable", "disable":
		if len(os.Args) != 3 {
			usage()
			os.Exit(2)
		}
		srv, ok := reg.Servers[os.Args[2]]
		if !ok {
			fatal(fmt.Errorf("unknown server %q", os.Args[2]))
		}
		v := os.Args[1] == "enable"
		srv.Enabled = &v
		reg.Servers[os.Args[2]] = srv
		mustSave(path, reg)
		fmt.Println(os.Args[1]+"d", os.Args[2])
	default:
		usage()
		os.Exit(2)
	}
}

func loadOrEmpty(path string) (config.Registry, error) {
	reg, err := config.LoadRegistry(path)
	if err == nil {
		return reg, nil
	}
	if os.IsNotExist(err) {
		return config.Registry{Servers: map[string]config.Server{}}, nil
	}
	return config.Registry{}, err
}
func mustSave(path string, reg config.Registry) {
	if err := save(path, reg); err != nil {
		fatal(err)
	}
}
func save(path string, reg config.Registry) error {
	for n, s := range reg.Servers {
		if err := s.Validate(n); err != nil {
			return err
		}
	}
	b, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
func fatal(err error) { fmt.Fprintln(os.Stderr, "error:", err); os.Exit(1) }
