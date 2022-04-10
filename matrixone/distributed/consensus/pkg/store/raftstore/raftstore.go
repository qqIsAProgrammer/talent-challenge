package raftstore

import (
	"errors"
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/logger"
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/store/raftstore/internal"
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/store/storage"
	"time"
)

var (
	ErrLostReadResponse = errors.New("it lost read response, retry read")
)

type RaftStore struct {
	pr *peer
}

func NewRaftStore(storeId uint64, dataPath string) *RaftStore {
	logger.SetLevel(logger.LogLevelInfo)
	rs := &RaftStore{pr: newPeer(storeId, dataPath)}
	return rs
}

func (rs *RaftStore) Set(key []byte, value []byte) error {
	cmd := internal.NewPutCmdRequest(key, value)
	return rs.pr.propose(cmd)
}

func (rs *RaftStore) Get(key []byte) ([]byte, error) {
	cb := rs.pr.linearizableRead(key)
	resp := cb.WaitRespWithTimeout(time.Second)
	if resp == nil {
		return nil, ErrLostReadResponse
	}
	value := resp.GetResponse().Get.Value
	if value == nil {
		return nil, storage.ErrNotFound
	}
	return value, nil
}

func (rs *RaftStore) Delete(key []byte) error {
	cmd := internal.NewDeleteCmdRequest(key)
	return rs.pr.propose(cmd)
}
