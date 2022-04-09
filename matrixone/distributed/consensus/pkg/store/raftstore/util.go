package raftstore

import (
	"go.etcd.io/etcd/raft/v3/raftpb"
)

func isEmptyConfState(confState raftpb.ConfState) bool {
	return len(confState.Voters) == 0 && len(confState.VotersOutgoing) == 0 &&
		len(confState.Learners) == 0 && len(confState.LearnersNext) == 0
}
