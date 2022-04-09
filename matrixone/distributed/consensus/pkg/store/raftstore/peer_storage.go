package raftstore

import (
	"github.com/golang/protobuf/proto"
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/logger"
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/store/raftstore/meta"
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/store/raftstore/raftstorepb"
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/store/raftstore/snap"
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/store/storage"
	"github.com/pkg/errors"
	"go.etcd.io/etcd/raft/v3"
	"go.etcd.io/etcd/raft/v3/raftpb"
)

const (
	defaultSnapTriedCnt = 5
)

type peerStorage struct {
	engines      *storage.Engines
	localState   *raftstorepb.RaftLocalState
	applyState   *raftstorepb.RaftApplyState
	confState    *raftpb.ConfState
	snapState    *snap.SnapState
	snapTriedCnt int
}

func newPeerStorage(path string) *peerStorage {
	ps := &peerStorage{
		engines: storage.NewEngines(path+storage.KvPath, path+storage.MetaPath),
	}
	ps.localState = meta.InitRaftLocalState(ps.engines)
	ps.applyState = meta.InitRaftApplyState(ps.engines)
	ps.confState = meta.InitRaftConfState(ps.engines)
	ps.snapState = &snap.SnapState{StateType: snap.SnapshotRelaxed}
	return ps
}

func (ps *peerStorage) InitialState() (raftpb.HardState, raftpb.ConfState, error) {
	hs := raftpb.HardState{}
	cs := raftpb.ConfState{}
	if !raft.IsEmptyHardState(*ps.localState.HardState) {
		hs = *ps.localState.HardState
	}
	if !isEmptyConfState(*ps.confState) {
		cs = *ps.confState
	}
	return hs, cs, nil
}

func (ps *peerStorage) Entries(lo, hi, maxSize uint64) ([]raftpb.Entry, error) {
	if err := ps.checkRange(lo, hi); err != nil || lo == hi {
		return nil, err
	}
	size := int(hi - lo)
	ents := make([]raftpb.Entry, 0, size)
	for i := lo; i < hi; i++ {
		key := meta.RaftLogEntryKey(i)
		val, err := ps.engines.ReadMeta(key)
		if err != nil {
			return nil, err
		}
		var ent raftpb.Entry
		if err = proto.Unmarshal(val, &ent); err != nil {
			return nil, err
		}
		if ent.Index != i {
			break
		}
		ents = append(ents, ent)
	}
	if len(ents) == size {
		return ents, nil
	}
	return nil, raft.ErrUnavailable
}

func (ps *peerStorage) Term(i uint64) (uint64, error) {
	if i == ps.truncateIndex() {
		return ps.truncateTerm(), nil
	}
	if err := ps.checkRange(i, i+1); err != nil {
		return 0, err
	}
	if i == ps.localState.LastIndex {
		return ps.localState.LastTerm, nil
	}
	key := meta.RaftLogEntryKey(i)
	val, err := ps.engines.ReadMeta(key)
	if err != nil {
		return 0, err
	}
	var ent raftpb.Entry
	if err = proto.Unmarshal(val, &ent); err != nil {
		return 0, err
	}
	return ent.Term, nil
}

func (ps *peerStorage) LastIndex() (uint64, error) {
	return ps.localState.LastIndex, nil
}

func (ps *peerStorage) FirstIndex() (uint64, error) {
	return ps.truncateIndex() + 1, nil
}

func (ps *peerStorage) Snapshot() (raftpb.Snapshot, error) {
	var snapshot raftpb.Snapshot
	if ps.snapState.StateType == snap.SnapshotGenerating {
		select {
		case s := <-ps.snapState.Receiver:
			if s != nil {
				snapshot = *s
			}
		default:
			return snapshot, raft.ErrSnapshotTemporarilyUnavailable
		}
		ps.snapState.StateType = snap.SnapshotRelaxed
		if !raft.IsEmptySnap(snapshot) {
			ps.snapTriedCnt = 0
			if ps.snapshotValid(snapshot) {
				return snapshot, nil
			}
		}
	}
	if ps.snapTriedCnt >= defaultSnapTriedCnt {
		ps.snapTriedCnt = 0
		return snapshot, errors.Errorf("failed to get snapshot after %d times", ps.snapTriedCnt)
	}

	ps.snapTriedCnt++
	ch := make(chan *raftpb.Snapshot, 1)
	ps.snapState = &snap.SnapState{
		StateType: snap.SnapshotGenerating,
		Receiver:  ch,
	}

	// schedule a snapshot generating task
	logger.Infof("leader is generating snapshot")
	go ps.SchedSnapGen(ch)

	return snapshot, raft.ErrSnapshotTemporarilyUnavailable
}

func (ps *peerStorage) AppliedIndex() uint64 {
	return ps.applyState.ApplyIndex
}

func (ps *peerStorage) truncateIndex() uint64 {
	return ps.applyState.TruncatedState.Index
}

func (ps *peerStorage) truncateTerm() uint64 {
	return ps.applyState.TruncatedState.Term
}

// append the entries to raft log and update localState
func (ps *peerStorage) truncateAndAppend(ents []raftpb.Entry) {
	if len(ents) == 0 {
		return
	}

	firsti, _ := ps.FirstIndex()
	lastEnt := ents[len(ents)-1]
	if firsti > lastEnt.Index {
		return
	}

	if firsti > ents[0].Index {
		ents = ents[ps.truncateIndex()-ents[0].Index+1:]
	}
	ps.appendLogEntries(ents)

	lasti, _ := ps.LastIndex()
	if lasti > lastEnt.Index {
		var deleted []raftpb.Entry
		for i := lastEnt.Index + 1; i <= lasti; i++ {
			deleted = append(deleted, raftpb.Entry{Index: i})
		}
		ps.deleteLogEntries(deleted)
	}

	ps.localState.LastTerm = lastEnt.Term
	ps.localState.LastIndex = lastEnt.Index
}

func (ps *peerStorage) SchedSnapGen(sender chan<- *raftpb.Snapshot) {
	term, _ := ps.Term(ps.AppliedIndex())
	metadata := raftpb.SnapshotMetadata{
		ConfState: *ps.confState,
		Index:     ps.AppliedIndex(),
		Term:      term,
	}
	snapshot := &raftpb.Snapshot{
		Data:     ps.engines.KVSnapshot(),
		Metadata: metadata,
	}
	sender <- snapshot
}

func (ps *peerStorage) applySnapshot(snapshot raftpb.Snapshot) {
	ps.localState.LastIndex = snapshot.Metadata.Index
	ps.localState.LastTerm = snapshot.Metadata.Term

	ps.applyState.ApplyIndex = snapshot.Metadata.Index
	ps.applyState.TruncatedState.Index = snapshot.Metadata.Index
	ps.applyState.TruncatedState.Term = snapshot.Metadata.Term
	ps.writeApplyState(ps.applyState)

	ps.confState = &snapshot.Metadata.ConfState
	ps.writeConfState(ps.confState)

	// applying snapshot
	logger.Infof("follower is applying snapshot")
	ps.snapState.StateType = snap.SnapshotApplying
	pairs := storage.Decode(snapshot.Data)
	for _, p := range pairs {
		modify := storage.PutData(p.Key, p.Val, true)
		ps.engines.WriteKV(modify)
	}
}

func (ps *peerStorage) saveReadyState(rd raft.Ready) error {
	if !raft.IsEmptySnap(rd.Snapshot) {
		ps.applySnapshot(rd.Snapshot)
	}
	ps.truncateAndAppend(rd.Entries)
	if !raft.IsEmptyHardState(rd.HardState) {
		ps.localState.HardState = &rd.HardState
	}
	return ps.writeLocalState(ps.localState)
}

func (ps *peerStorage) deleteLogEntries(ents []raftpb.Entry) error {
	for _, ent := range ents {
		key := meta.RaftLogEntryKey(ent.Index)
		if err := ps.deleteMeta(key, &ent); err != nil {
			return err
		}
	}
	return nil
}

func (ps *peerStorage) appendLogEntries(ents []raftpb.Entry) error {
	for _, ent := range ents {
		key := meta.RaftLogEntryKey(ent.Index)
		if err := ps.putMeta(key, &ent); err != nil {
			return err
		}
	}
	return nil
}

func (ps *peerStorage) writeLocalState(localState *raftstorepb.RaftLocalState) error {
	return ps.putMeta(meta.RaftLocalStateKey(), localState)
}

func (ps *peerStorage) writeApplyState(applyState *raftstorepb.RaftApplyState) error {
	return ps.putMeta(meta.RaftApplyStateKey(), applyState)
}

func (ps *peerStorage) writeConfState(confState *raftpb.ConfState) error {
	return ps.putMeta(meta.RaftConfStateKey(), confState)
}

func (ps *peerStorage) putMeta(key []byte, msg proto.Message) error {
	modify := storage.PutMeta(key, msg, true)
	return ps.engines.WriteMeta(modify)
}

func (ps *peerStorage) deleteMeta(key []byte, msg proto.Message) error {
	modify := storage.DeleteMeta(key, true)
	return ps.engines.WriteMeta(modify)
}

func (ps *peerStorage) checkRange(lo, hi uint64) error {
	if lo > hi {
		return errors.Errorf("range error: low %d is greater than high %d", lo, hi)
	} else if hi > ps.localState.LastIndex+1 {
		return errors.Errorf("range error: high %d is out of bound", hi)
	} else if lo <= ps.truncateIndex() {
		return raft.ErrCompacted
	}
	return nil
}

func (ps *peerStorage) snapshotValid(snapshot raftpb.Snapshot) bool {
	return snapshot.Metadata.Index >= ps.truncateIndex()
}
