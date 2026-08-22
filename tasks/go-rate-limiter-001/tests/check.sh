#!/usr/bin/env bash
set -Eeuo pipefail
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
cp /solution/main.go "$tmp/main.go"
cat > "$tmp/main_test.go" <<'GO'
package main

import (
  "testing"
  "time"
)

func TestBurst(t *testing.T) {
  r := NewRateLimiter(2, 1)
  now := time.Unix(0, 0)
  if !r.Allow("a", now) || !r.Allow("a", now) || r.Allow("a", now) { t.Fatal("burst policy is wrong") }
}

func TestRefill(t *testing.T) {
  r := NewRateLimiter(1, 1)
  now := time.Unix(0, 0)
  if !r.Allow("a", now) || r.Allow("a", now) || !r.Allow("a", now.Add(time.Second)) { t.Fatal("refill policy is wrong") }
}

func TestKeysIndependent(t *testing.T) {
  r := NewRateLimiter(1, 1)
  now := time.Unix(0, 0)
  if !r.Allow("a", now) || !r.Allow("b", now) { t.Fatal("keys share state") }
}

func TestEmpty(t *testing.T) {
  r := NewRateLimiter(1, 0)
  now := time.Unix(0, 0)
  if !r.Allow("a", now) || r.Allow("a", now.Add(time.Hour)) { t.Fatal("empty bucket was accepted") }
}
GO
(cd "$tmp" && GO111MODULE=off go test)
