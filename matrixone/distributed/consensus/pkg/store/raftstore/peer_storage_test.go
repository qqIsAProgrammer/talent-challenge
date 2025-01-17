package raftstore

import (
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/store/raftstore/meta"
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/store/raftstore/raftstorepb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.etcd.io/etcd/raft/v3"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"testing"
	"time"
)

const (
	enginePath = "../test_data"
)

func newTestPeerStorage() *peerStorage {
	return newPeerStorage(enginePath)
}

func newTestPeerStorageFromEntries(t *testing.T, entries []raftpb.Entry) *peerStorage {
	ps := newTestPeerStorage()
	ps.truncateAndAppend(entries[1:])
	applyState := ps.applyState
	applyState.TruncatedState = &raftstorepb.RaftTruncatedState{
		Term:  entries[0].Term,
		Index: entries[0].Index,
	}
	applyState.ApplyIndex = entries[len(entries)-1].Index
	err := ps.writeLocalState(ps.localState)
	require.Nil(t, err)
	err = ps.appendLogEntries(entries)
	require.Nil(t, err)
	err = ps.writeApplyState(ps.applyState)
	require.Nil(t, err)
	return ps
}

func cleanUpData(ps *peerStorage) {
	if err := ps.engines.Destroy(); err != nil {
		panic(err)
	}
}

func newTestEntry(term, index uint64) raftpb.Entry {
	return raftpb.Entry{
		Term:  term,
		Index: index,
		Data:  []byte("0"),
	}
}

func TestPeerStorageTerm(t *testing.T) {
	entries := []raftpb.Entry{
		newTestEntry(3, 3),
		newTestEntry(4, 4),
		newTestEntry(5, 5),
	}
	testDatas := []struct {
		term  uint64
		index uint64
		err   error
	}{
		{1, 2, raft.ErrCompacted},
		{3, 3, nil},
		{4, 4, nil},
		{5, 5, nil},
	}
	for _, testData := range testDatas {
		ps := newTestPeerStorageFromEntries(t, entries)
		term, err := ps.Term(testData.index)
		if err != nil {
			assert.Equal(t, testData.err, err)
		} else {
			assert.Equal(t, testData.term, term)
		}
		cleanUpData(ps)
	}
}

func TestPeerStorageFirstIndex(t *testing.T) {
	entries := []raftpb.Entry{
		newTestEntry(3, 3),
		newTestEntry(4, 4),
		newTestEntry(5, 5),
	}
	ps := newTestPeerStorageFromEntries(t, entries)
	firstIndex, err := ps.FirstIndex()
	assert.Nil(t, err)
	assert.Equal(t, uint64(4), firstIndex)
	cleanUpData(ps)
}

func TestPeerStorageLastIndex(t *testing.T) {
	entries := []raftpb.Entry{
		newTestEntry(3, 3),
		newTestEntry(4, 4),
		newTestEntry(5, 5),
	}
	ps := newTestPeerStorageFromEntries(t, entries)
	lastIndex, err := ps.LastIndex()
	assert.Nil(t, err)
	assert.Equal(t, uint64(5), lastIndex)
	cleanUpData(ps)
}

func TestPeerStorageEntries(t *testing.T) {
	entries := []raftpb.Entry{
		newTestEntry(3, 3),
		newTestEntry(4, 4),
		newTestEntry(5, 5),
		newTestEntry(6, 6),
		newTestEntry(7, 7),
	}
	testDatas := []struct {
		lo   uint64
		hi   uint64
		ents []raftpb.Entry
		err  error
	}{
		{3, 4, nil, raft.ErrCompacted},
		{4, 5, []raftpb.Entry{newTestEntry(4, 4)}, nil},
		{4, 8, []raftpb.Entry{
			newTestEntry(4, 4),
			newTestEntry(5, 5),
			newTestEntry(6, 6),
			newTestEntry(7, 7),
		}, nil},
	}
	const maxSize = 0
	for _, testData := range testDatas {
		ps := newTestPeerStorageFromEntries(t, entries)
		// truncated.Index = 3, firstIndex = 4, lastIndex = 7,
		ents, err := ps.Entries(testData.lo, testData.hi, maxSize)
		if err != nil {
			assert.Equal(t, testData.err, err)
		} else {
			assert.Equal(t, testData.ents, ents)
		}
		cleanUpData(ps)
	}
}

func TestPeerStorageAppliedIndex(t *testing.T) {
	entries := []raftpb.Entry{
		newTestEntry(3, 3),
		newTestEntry(4, 4),
		newTestEntry(5, 5),
	}
	ps := newTestPeerStorageFromEntries(t, entries)
	defer cleanUpData(ps)
	applyIndex := ps.AppliedIndex()
	assert.Equal(t, uint64(5), applyIndex)
}

func TestPeerStorageSnapshot(t *testing.T) {
	// TODO: do this test function after finishing another todo in peerStorage.Snapshot()
}

func TestPeerStorageAppendAndUpdate(t *testing.T) {
	entries := []raftpb.Entry{
		newTestEntry(3, 3),
		newTestEntry(4, 4),
		newTestEntry(5, 5),
	}
	testDatas := []struct {
		append []raftpb.Entry
		result []raftpb.Entry
	}{
		{
			[]raftpb.Entry{
				newTestEntry(3, 3),
				newTestEntry(4, 4),
			},
			[]raftpb.Entry{
				newTestEntry(4, 4),
			},
		},
		{
			[]raftpb.Entry{
				newTestEntry(3, 3),
				newTestEntry(6, 4),
				newTestEntry(6, 5),
			},
			[]raftpb.Entry{
				newTestEntry(6, 4),
				newTestEntry(6, 5),
			},
		},
		{
			[]raftpb.Entry{
				newTestEntry(3, 3),
				newTestEntry(4, 4),
				newTestEntry(5, 5),
				newTestEntry(5, 6),
			},
			[]raftpb.Entry{
				newTestEntry(4, 4),
				newTestEntry(5, 5),
				newTestEntry(5, 6),
			},
		},
		{
			[]raftpb.Entry{
				newTestEntry(3, 2),
				newTestEntry(3, 3),
				newTestEntry(5, 4),
			},

			[]raftpb.Entry{
				newTestEntry(5, 4),
			},
		},
		{
			[]raftpb.Entry{
				newTestEntry(5, 4),
			},
			[]raftpb.Entry{
				newTestEntry(5, 4),
			},
		},
		{
			[]raftpb.Entry{
				newTestEntry(5, 6),
			},
			[]raftpb.Entry{
				newTestEntry(4, 4),
				newTestEntry(5, 5),
				newTestEntry(5, 6),
			},
		},
	}
	for _, testData := range testDatas {
		ps := newTestPeerStorageFromEntries(t, entries)
		ps.truncateAndAppend(testData.append)
		const maxSize = 0
		firstIndex, _ := ps.FirstIndex()
		lastIndex, _ := ps.LastIndex()
		result, err := ps.Entries(firstIndex, lastIndex+1, maxSize)
		assert.Nil(t, err)
		assert.Equal(t, testData.result, result)
		cleanUpData(ps)
	}
}

func TestPeerStorageRestart(t *testing.T) {
	entries := []raftpb.Entry{
		newTestEntry(3, 3),
		newTestEntry(4, 4),
		newTestEntry(5, 5),
	}
	// Start peer storage with given entries
	ps := newTestPeerStorageFromEntries(t, entries)
	assert.Nil(t, ps.engines.Close())
	// Restart peer storage without given entries
	time.Sleep(1)
	ps = newTestPeerStorage()
	assert.Equal(t, uint64(5), ps.localState.LastTerm)
	assert.Equal(t, uint64(5), ps.localState.LastIndex)
	assert.Equal(t, uint64(5), ps.applyState.ApplyIndex)
	assert.Equal(t, uint64(3), ps.applyState.TruncatedState.Term)
	assert.Equal(t, uint64(3), ps.applyState.TruncatedState.Index)
	for index := 3; index <= 5; index++ {
		key := meta.RaftLogEntryKey(uint64(index))
		val, err := ps.engines.ReadMeta(key)
		assert.Nil(t, err)
		var entry raftpb.Entry
		assert.Nil(t, entry.Unmarshal(val))
		assert.Equal(t, newTestEntry(uint64(index), uint64(index)), entry)
	}
}
