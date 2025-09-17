// Copyright 2025 Antenore Gatta
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sessions

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Manager struct {
	db *sql.DB
	dbPath string
}

type Message struct {
	ID        int64
	SessionID int64
	Role      string
	Content   string
	Timestamp time.Time
}

type Session struct {
	ID        int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	dbDir := filepath.Join(home, ".deecli")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	dbPath := filepath.Join(dbDir, "session.db")
	
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	m := &Manager{
		db:     db,
		dbPath: dbPath,
	}

	if err := m.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return m, nil
}

func (m *Manager) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id INTEGER NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (session_id) REFERENCES sessions(id)
	);

	CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id);
	`

	_, err := m.db.Exec(schema)
	return err
}

func (m *Manager) GetCurrentSession() (*Session, error) {
	var session Session
	err := m.db.QueryRow(`
		SELECT id, created_at, updated_at 
		FROM sessions 
		ORDER BY updated_at DESC 
		LIMIT 1
	`).Scan(&session.ID, &session.CreatedAt, &session.UpdatedAt)

	if err == sql.ErrNoRows {
		return m.CreateSession()
	}
	if err != nil {
		return nil, err
	}

	return &session, nil
}

func (m *Manager) CreateSession() (*Session, error) {
	result, err := m.db.Exec(`
		INSERT INTO sessions (created_at, updated_at) 
		VALUES (CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Session{
		ID:        id,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (m *Manager) SaveMessage(sessionID int64, role, content string) error {
	_, err := m.db.Exec(`
		INSERT INTO messages (session_id, role, content, timestamp)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	`, sessionID, role, content)

	if err != nil {
		return err
	}

	_, err = m.db.Exec(`
		UPDATE sessions 
		SET updated_at = CURRENT_TIMESTAMP 
		WHERE id = ?
	`, sessionID)

	return err
}

func (m *Manager) GetSessionMessages(sessionID int64) ([]Message, error) {
	rows, err := m.db.Query(`
		SELECT id, session_id, role, content, timestamp
		FROM messages
		WHERE session_id = ?
		ORDER BY timestamp ASC
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		err := rows.Scan(&msg.ID, &msg.SessionID, &msg.Role, &msg.Content, &msg.Timestamp)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

func (m *Manager) HasPreviousSession() bool {
	var count int
	err := m.db.QueryRow(`
		SELECT COUNT(*) FROM sessions
	`).Scan(&count)
	
	return err == nil && count > 0
}

func (m *Manager) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}