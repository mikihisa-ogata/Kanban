package main

import "testing"

func TestListenAddr(t *testing.T) {
	t.Setenv("LISTEN_ADDR", "")
	if got := listenAddr(); got != "localhost:8080" {
		t.Errorf("listenAddr() = %q, want %q", got, "localhost:8080")
	}

	t.Setenv("LISTEN_ADDR", ":8080")
	if got := listenAddr(); got != ":8080" {
		t.Errorf("listenAddr() = %q, want %q", got, ":8080")
	}
}
