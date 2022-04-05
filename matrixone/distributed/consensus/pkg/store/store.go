package store

import (
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/cfg"
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/store/raftstore"
	"strings"
)

// Store the store interface
type Store interface {
	// Set set key-value to store
	Set(key []byte, value []byte) error
	// Get returns the value from store
	Get(key []byte) ([]byte, error)
	// Delete remove the key from store
	Delete(key []byte) error
}

// NewStore create the raft store
func NewStore(cfg cfg.StoreCfg) (Store, error) {
	if cfg.Memory {
		return newMemoryStore()
	}

	// TODO: need to implement
	dirs := strings.Split(cfg.DataPath, "/")
	id := nodeMap[dirs[len(dirs)-1]]
	return raftstore.NewRaftStore(id, cfg.DataPath), nil
}

var nodeMap = map[string]uint64{
	"node1": 1,
	"node2": 2,
	"node3": 3,
}
