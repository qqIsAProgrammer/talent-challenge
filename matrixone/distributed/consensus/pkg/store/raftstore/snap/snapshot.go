package snap

import "go.etcd.io/etcd/raft/v3/raftpb"

type SnapStateType int

const (
	SnapshotGenerating SnapStateType = iota
	SnapshotRelaxed
	SnapshotApplying
)

type SnapState struct {
	StateType SnapStateType
	Receiver  chan *raftpb.Snapshot
}
