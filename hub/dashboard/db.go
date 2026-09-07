package main

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;
CREATE TABLE IF NOT EXISTS users(
  id INTEGER PRIMARY KEY, username TEXT UNIQUE NOT NULL, password_hash TEXT NOT NULL,
  role TEXT NOT NULL DEFAULT 'user', created_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS sessions(token_hash TEXT PRIMARY KEY, user_id INTEGER NOT NULL, expires_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS projects(
  id INTEGER PRIMARY KEY, name TEXT NOT NULL, slug TEXT UNIQUE NOT NULL, project_key TEXT UNIQUE NOT NULL,
  timezone TEXT NOT NULL DEFAULT 'America/Toronto', dashboard TEXT NOT NULL DEFAULT '[]', created_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS project_members(project_id INTEGER NOT NULL, user_id INTEGER NOT NULL, PRIMARY KEY(project_id,user_id));
CREATE TABLE IF NOT EXISTS devices(
  id INTEGER PRIMARY KEY, project_id INTEGER NOT NULL, slug TEXT NOT NULL, name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending', token_hash TEXT UNIQUE NOT NULL,
  hostname TEXT DEFAULT '', os TEXT DEFAULT '', arch TEXT DEFAULT '', agent_version TEXT DEFAULT '', tailnet_ip TEXT DEFAULT '',
  config TEXT NOT NULL DEFAULT '{}', config_version INTEGER NOT NULL DEFAULT 1,
  headscale_key TEXT DEFAULT '', headscale_key_delivered INTEGER NOT NULL DEFAULT 0,
  last_heartbeat TEXT DEFAULT '', last_seen TEXT DEFAULT '', enrolled_at TEXT NOT NULL, approved_at TEXT DEFAULT '',
  UNIQUE(project_id, slug));
CREATE TABLE IF NOT EXISTS actions(
  id INTEGER PRIMARY KEY, project_id INTEGER NOT NULL, name TEXT NOT NULL, kind TEXT NOT NULL DEFAULT 'hub',
  device_id INTEGER, method TEXT NOT NULL DEFAULT 'GET', url TEXT NOT NULL, headers TEXT NOT NULL DEFAULT '{}',
  body TEXT NOT NULL DEFAULT '', timeout_seconds INTEGER NOT NULL DEFAULT 10, last_run TEXT DEFAULT '', last_result TEXT DEFAULT '');
CREATE TABLE IF NOT EXISTS automations(
  id INTEGER PRIMARY KEY, project_id INTEGER NOT NULL, name TEXT NOT NULL, enabled INTEGER NOT NULL DEFAULT 1,
  cron TEXT NOT NULL, action_id INTEGER NOT NULL, last_run TEXT DEFAULT '', last_result TEXT DEFAULT '');
CREATE TABLE IF NOT EXISTS commands(
  id INTEGER PRIMARY KEY, device_id INTEGER NOT NULL, kind TEXT NOT NULL, payload TEXT NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'queued', result TEXT DEFAULT '', created_at TEXT NOT NULL, finished_at TEXT DEFAULT '',
  action_id INTEGER, automation_id INTEGER);
CREATE INDEX IF NOT EXISTS idx_commands_device ON commands(device_id, status);
`

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4) // WAL : lecteurs concurrents ; busy_timeout gère l'écrivain unique
	migrate(db)
	for _, stmt := range strings.Split(schema, ";\n") {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			return nil, fmt.Errorf("schéma : %w (%s)", err, strings.TrimSpace(stmt)[:40])
		}
	}
	return db, nil
}

// migrate : renommages de colonnes des versions précédentes.
func migrate(db *sql.DB) {
	rows, err := db.Query(`PRAGMA table_info(devices)`)
	if err != nil {
		return
	}
	hasScreenID := false
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dflt any
		rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk)
		if name == "screen_id" {
			hasScreenID = true
		}
	}
	rows.Close()
	if hasScreenID {
		db.Exec(`ALTER TABLE devices RENAME COLUMN screen_id TO slug`)
	}
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }

func randomToken(prefix string, bytes int) string {
	b := make([]byte, bytes)
	rand.Read(b)
	return prefix + hex.EncodeToString(b)
}

func hashToken(t string) string {
	h := sha256.Sum256([]byte(t))
	return hex.EncodeToString(h[:])
}

func slugify(s string) string {
	var b strings.Builder
	last := '-'
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			last = r
		default:
			if last != '-' {
				b.WriteRune('-')
				last = '-'
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
