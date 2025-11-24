package store

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/CrowGreen/dkv/internal/crypto"
)

type snapshot struct {
	Data    map[string][]byte `json:"data"`
	Version int64             `json:"version"`
}

// SaveSnapshot encrypts and writes store state to disk
func (s *Store) SaveSnapshot(path string, enc *crypto.Encryptor) error {
	s.mu.RLock()
	snap := snapshot{
		Data:    make(map[string][]byte),
		Version: s.version,
	}
	for k, v := range s.data {
		snap.Data[k] = v.Value
	}
	s.mu.RUnlock()

	raw, err := json.Marshal(snap)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}

	encrypted, err := enc.Encrypt(raw)
	if err != nil {
		return fmt.Errorf("encrypt snapshot: %w", err)
	}

	return os.WriteFile(path, encrypted, 0600)
}

// LoadSnapshot decrypts and restores store state from disk
func (s *Store) LoadSnapshot(path string, enc *crypto.Encryptor) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no snapshot yet
		}
		return fmt.Errorf("read snapshot: %w", err)
	}

	decrypted, err := enc.Decrypt(raw)
	if err != nil {
		return fmt.Errorf("decrypt snapshot: %w", err)
	}

	var snap snapshot
	if err := json.Unmarshal(decrypted, &snap); err != nil {
		return fmt.Errorf("unmarshal snapshot: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.version = snap.Version
	for k, v := range snap.Data {
		s.data[k] = &Entry{Value: v, Version: snap.Version}
	}
	return nil
}
