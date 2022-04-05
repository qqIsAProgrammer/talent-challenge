package raftstore

import (
	"encoding/binary"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"time"
)

func buildReadCtx(key []byte) []byte {
	buf := make([]byte, 8)
	ts := time.Now().UnixNano()
	binary.LittleEndian.PutUint64(buf, uint64(ts))
	return append(buf, key...)
}

func isEmptyConfState(confState raftpb.ConfState) bool {
	return len(confState.Voters) == 0 && len(confState.VotersOutgoing) == 0 &&
		len(confState.Learners) == 0 && len(confState.LearnersNext) == 0
}

func isEmptyHardState(hardState raftpb.HardState) bool {
	empty := raftpb.HardState{}
	return hardState.Term == empty.Term && hardState.Vote == empty.Vote &&
		hardState.Commit == empty.Commit
}

func isEmptySnapshotMetadata(meta raftpb.SnapshotMetadata) bool {
	return meta.Index == 0 && meta.Term == 0 && isEmptyConfState(meta.ConfState)
}