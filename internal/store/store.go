package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/tanq16/claude-usage/internal/model"
)

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "claude-usage", "tasks.json")
}

type Store struct {
	path  string
	mu    sync.Mutex
	tasks map[string]model.Task
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}

	s := &Store{path: path, tasks: make(map[string]model.Task)}

	data, err := os.ReadFile(path)
	if err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &s.tasks); err != nil {
			return nil, fmt.Errorf("parse tasks file: %w", err)
		}
	}
	// If file doesn't exist, start with empty map — no error

	return s, nil
}

func (s *Store) Close() error {
	return nil
}

func (s *Store) flush() error {
	data, err := json.MarshalIndent(s.tasks, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

func generateID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Store) AddTask(task model.Task) (model.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task.ID = generateID()
	s.tasks[task.ID] = task
	return task, s.flush()
}

func (s *Store) ListTasks() ([]model.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tasks := make([]model.Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (s *Store) UpdateTask(id string, fn func(*model.Task)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[id]
	if !ok {
		return fmt.Errorf("task %s not found", id)
	}
	fn(&t)
	s.tasks[id] = t
	return s.flush()
}

func (s *Store) DeleteTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[id]; !ok {
		return fmt.Errorf("task %s not found", id)
	}
	delete(s.tasks, id)
	return s.flush()
}
