package storage

import (
	"errors"
	"github.com/cockroachdb/pebble"
	"os"
)

var (
	ErrUnknownModify = errors.New("unknown modify type")
	ErrNotFound      = errors.New("key does not exist")
)

const (
	KvPath   = "/kv"
	MetaPath = "/meta"
)

type Engines struct {
	// kv stores user data and kv meta
	kv     *storage
	kvPath string
	// meta stores raft meta
	meta     *storage
	metaPath string
}

func NewEngines(kvPath, raftPath string) *Engines {
	opts := (&pebble.Options{}).EnsureDefaults()
	return &Engines{
		kv:       newStorage(kvPath, opts),
		kvPath:   kvPath,
		meta:     newStorage(raftPath, opts),
		metaPath: raftPath,
	}
}

func (e *Engines) WriteKV(m Modify) error {
	switch m.Data.(type) {
	case Put:
		return e.kv.set(m.Key(), m.Value(), m.Sync())
	case Delete:
		return e.kv.delete(m.Key(), m.Sync())
	default:
		return ErrUnknownModify
	}
}

func (e *Engines) WriteMeta(m Modify) error {
	switch m.Data.(type) {
	case Put:
		return e.meta.set(m.Key(), m.Value(), m.Sync())
	case Delete:
		return e.meta.delete(m.Key(), m.Sync())
	default:
		return ErrUnknownModify
	}
}

func (e *Engines) ReadKV(key []byte) ([]byte, error) {
	value, err := e.kv.get(key)
	if err == pebble.ErrNotFound {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (e *Engines) ReadMeta(key []byte) ([]byte, error) {
	value, err := e.meta.get(key)
	if err == pebble.ErrNotFound {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (e *Engines) KVSnapshot() []byte {
	return e.kv.snapshot()
}

func (e *Engines) Close() error {
	if err := e.kv.close(); err != nil {
		return err
	}
	if err := e.meta.close(); err != nil {
		return err
	}
	return nil
}

func (e *Engines) Destroy() error {
	if err := e.Close(); err != nil {
		return err
	}
	if err := os.RemoveAll(e.kvPath); err != nil {
		return err
	}
	if err := os.RemoveAll(e.metaPath); err != nil {
		return err
	}
	return nil
}
