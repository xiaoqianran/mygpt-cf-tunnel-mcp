package config

import "testing"

func TestToolAllowed(t *testing.T) {
	tests := []struct {
		name string
		srv  Server
		tool string
		want bool
	}{
		{"default allow", Server{}, "read_docs", true},
		{"exact allow", Server{AllowTools: []string{"read_docs"}}, "read_docs", true},
		{"not allowed", Server{AllowTools: []string{"read_docs"}}, "delete_repo", false},
		{"prefix allow", Server{AllowTools: []string{"read_*"}}, "read_docs", true},
		{"deny wins", Server{AllowTools: []string{"*"}, DenyTools: []string{"delete_*"}}, "delete_repo", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.srv.ToolAllowed(tt.tool); got != tt.want {
				t.Fatalf("ToolAllowed(%q)=%v want %v", tt.tool, got, tt.want)
			}
		})
	}
}

func TestTransportName(t *testing.T) {
	if got := (Server{URL: "https://example.com/mcp"}).TransportName(); got != "streamable_http" {
		t.Fatal(got)
	}
	if got := (Server{Transport: "streamable-http"}).TransportName(); got != "streamable_http" {
		t.Fatal(got)
	}
	if got := (Server{Transport: "stdio"}).TransportName(); got != "stdio" {
		t.Fatal(got)
	}
}
