package meta

import (
	"github.com/golang/protobuf/proto"
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/store/raftstore/raftstorepb"
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/store/storage"
	"go.etcd.io/etcd/raft/v3/raftpb"
)

func GetRaftLocalState(engines *storage.Engines) (*raftstorepb.RaftLocalState, error) {
	localState := &raftstorepb.RaftLocalState{
		HardState: &raftpb.HardState{},
	}
	value, err := engines.ReadMeta(RaftLocalStateKey())
	if err == storage.ErrNotFound {
		return localState, nil
	}
	if err != nil {
		return nil, err
	}
	if err = proto.Unmarshal(value, localState); err != nil {
		return nil, err
	}
	return localState, nil
}

func InitRaftLocalState(engines *storage.Engines) *raftstorepb.RaftLocalState {
	localState, err := GetRaftLocalState(engines)
	if err != nil {
		panic(err)
	}
	return localState
}

func GetRaftApplyState(engines *storage.Engines) (*raftstorepb.RaftApplyState, error) {
	applyState := &raftstorepb.RaftApplyState{
		TruncatedState: &raftstorepb.RaftTruncatedState{},
	}
	value, err := engines.ReadMeta(RaftApplyStateKey())
	if err == storage.ErrNotFound {
		return applyState, nil
	}
	if err != nil {
		return nil, err
	}
	if err = proto.Unmarshal(value, applyState); err != nil {
		return nil, err
	}
	return applyState, nil
}

func InitRaftApplyState(engines *storage.Engines) *raftstorepb.RaftApplyState {
	applyState, err := GetRaftApplyState(engines)
	if err != nil {
		panic(err)
	}
	return applyState
}

func GetRaftConfState(engines *storage.Engines) (*raftpb.ConfState, error) {
	confState := &raftpb.ConfState{}
	value, err := engines.ReadMeta(RaftConfStateKey())
	if err == storage.ErrNotFound {
		return confState, nil
	}
	if err != nil {
		return nil, err
	}
	if err = proto.Unmarshal(value, confState); err != nil {
		return nil, err
	}
	return confState, nil
}

func InitRaftConfState(engines *storage.Engines) *raftpb.ConfState {
	confState, err := GetRaftConfState(engines)
	if err != nil {
		panic(err)
	}
	return confState
}
