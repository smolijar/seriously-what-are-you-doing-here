package main

import (
	"net/http"
	"testing"
)

func TestCookieValue(t *testing.T) {
	value, err := cookieValue([]*http.Cookie{{Name: "x", Value: "1"}, {Name: "d", Value: "secret"}}, "d")
	if err != nil {
		t.Fatalf("cookieValue returned error: %v", err)
	}
	if value != "secret" {
		t.Fatalf("expected secret, got %q", value)
	}
}

func TestCookieValueMissing(t *testing.T) {
	if _, err := cookieValue([]*http.Cookie{{Name: "x", Value: "1"}}, "d"); err == nil {
		t.Fatal("expected error when cookie is missing")
	}
}
