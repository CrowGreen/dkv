package wal

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
)

// entry format: [len(4 bytes)][op(1 byte)][keyLen(2 bytes)][key][value]

const (
	OpPut    byte = 0x01
	OpDelete byte = 0x02
)

type Entry struct {
	Op    byte
	Key   string
	Value []byte
}

type WAL struct {
	mu   sync.Mutex
	file *os.File
	path string
}

func Open(path string) (*WAL, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("wal open: %w", err)
	}
	return &WAL{file: f, path: path}, nil
}

func (w *WAL) Append(entry Entry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	keyBytes := []byte(entry.Key)
	// total = 1 (op) + 2 (key len) + len(key) + len(value)
	totalLen := 1 + 2 + len(keyBytes) + len(entry.Value)

	buf := make([]byte, 4+totalLen)
	binary.BigEndian.PutUint32(buf[0:4], uint32(totalLen))
	buf[4] = entry.Op
	binary.BigEndian.PutUint16(buf[5:7], uint16(len(keyBytes)))
	copy(buf[7:7+len(keyBytes)], keyBytes)
	copy(buf[7+len(keyBytes):], entry.Value)

	_, err := w.file.Write(buf)
	if err != nil {
		return fmt.Errorf("wal write: %w", err)
	}
	return w.file.Sync()
}

func (w *WAL) Replay() ([]Entry, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	var entries []Entry
	for {
		var lenBuf [4]byte
		if _, err := io.ReadFull(w.file, lenBuf[:]); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return nil, err
		}
		totalLen := binary.BigEndian.Uint32(lenBuf[:])
		data := make([]byte, totalLen)
		if _, err := io.ReadFull(w.file, data); err != nil {
			return nil, fmt.Errorf("wal read entry: %w", err)
		}

		op := data[0]
		keyLen := binary.BigEndian.Uint16(data[1:3])
		key := string(data[3 : 3+keyLen])
		value := data[3+keyLen:]

		entries = append(entries, Entry{Op: op, Key: key, Value: value})
	}
	return entries, nil
}

func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Close()
}

func (w *WAL) Truncate() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.file.Truncate(0); err != nil {
		return err
	}
	_, err := w.file.Seek(0, io.SeekStart)
	return err
}
