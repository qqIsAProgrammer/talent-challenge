package raftstore

import (
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/logger"
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/store/raftstore/internal"
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/store/raftstore/raftstorepb"
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/store/storage"
	"go.etcd.io/etcd/raft/v3"
	"go.etcd.io/etcd/raft/v3/quorum"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"math"
	"net"
	"time"
)

const (
	defaultLogGCCountLimit    = 1000
	defaultCompactCheckPeriod = 100
)

var peerMap = map[uint64]string{
	1: "127.0.0.1:6060",
	2: "127.0.0.1:6061",
	3: "127.0.0.1:6062",
}

func loadRaftPeers() []raft.Peer {
	rpeers := make([]raft.Peer, len(peerMap))
	for i := range rpeers {
		rpeers[i] = raft.Peer{ID: uint64(i + 1)}
	}
	return rpeers
}

type read struct {
	ctx []byte
	cb  *internal.Callback
}

func (r *read) key() []byte {
	return r.ctx[8:]
}

type peer struct {
	id        uint64
	raftGroup raft.Node
	ps        *peerStorage
	router    *router

	readc           chan *read
	readStateTable  map[string]raft.ReadState
	readStateComing chan struct{}

	recvc chan raftpb.Message

	compactionElapse  int
	compactionTimeout int

	lastCompactedIdx uint64
}

func newPeer(id uint64, path string) *peer {
	pr := &peer{
		id:                id,
		ps:                newPeerStorage(path),
		readc:             make(chan *read, 1024),
		readStateTable:    make(map[string]raft.ReadState),
		readStateComing:   make(chan struct{}, 1),
		recvc:             make(chan raftpb.Message, 1024),
		compactionTimeout: defaultCompactCheckPeriod,
	}
	pr.router = newRouter(peerMap[id], pr.recvc)
	pr.lastCompactedIdx = pr.ps.truncateIndex()

	c := &raft.Config{
		ID:                        id,
		ElectionTick:              10,
		HeartbeatTick:             1,
		Storage:                   pr.ps,
		Applied:                   pr.ps.AppliedIndex(),
		MaxSizePerMsg:             1024 * 1024,
		MaxUncommittedEntriesSize: 1 << 30,
		MaxInflightMsgs:           256,
		PreVote:                   true,
	}
	if pr.bootstrap() {
		pr.raftGroup = raft.StartNode(c, loadRaftPeers())
		pr.ps.confState = pr.confState()
		pr.ps.writeConfState(pr.ps.confState)
	} else {
		pr.raftGroup = raft.RestartNode(c)
	}
	logger.Infof("etcd raft is started, node: %d", id)

	pr.run()
	return pr
}

func (pr *peer) propose(cmd *raftstorepb.RaftCmdRequest) error {
	data, err := proto.Marshal(cmd)
	if err != nil {
		return err
	}
	return pr.raftGroup.Propose(context.TODO(), data)
}

func (pr *peer) run() {
	go pr.serveGrpc(pr.id)
	go pr.onTick()
	go pr.handleRaftMsgs()
	go pr.handleReadState()
}

func (pr *peer) serveGrpc(id uint64) {
	l, err := net.Listen("tcp", peerMap[id])
	if err != nil {
		panic(err)
	}

	logger.Infof("%d listen %v success\n", id, peerMap[id])
	srv := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 15 * time.Second,
			Time:              5 * time.Second,
			Timeout:           1 * time.Second,
		}),
	)

	raftstorepb.RegisterMessageServer(srv, pr.router.raftServer)
	srv.Serve(l)
}

func (pr *peer) onTick() {
	ticker := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			pr.tick()
		case rd := <-pr.raftGroup.Ready():
			pr.handleReady(rd)
		}
	}
}

func (pr *peer) tick() {
	pr.raftGroup.Tick()
	pr.tickLogGC()
}

func (pr *peer) tickLogGC() {
	pr.compactionElapse++
	if pr.compactionElapse >= pr.compactionTimeout {
		pr.compactionElapse = 0
		// try to compact log
		pr.onLogGCTask()
	}
}

func (pr *peer) onLogGCTask() {
	if !pr.isLeader() {
		return
	}

	appliedIdx := pr.ps.AppliedIndex()
	firstIdx, _ := pr.ps.FirstIndex()
	var compactIdx uint64
	if appliedIdx > firstIdx && appliedIdx-firstIdx >= defaultLogGCCountLimit {
		compactIdx = appliedIdx
	} else {
		return
	}

	// improve the success rate of log compaction
	compactIdx--
	term, err := pr.ps.Term(compactIdx)
	if err != nil {
		logger.Fatalf("appliedIdx: %d, firstIdx: %d, compactIdx: %d", appliedIdx, firstIdx, compactIdx)
		panic(err)
	}

	pr.propose(internal.NewCompactCmdRequest(compactIdx, term))
}

func (pr *peer) handleRaftMsgs() {
	for {
		msgs := make([]raftpb.Message, 0)
		select {
		case msg := <-pr.recvc:
			msgs = append(msgs, msg)
		}
		pending := len(pr.recvc)
		for i := 0; i < pending; i++ {
			msgs = append(msgs, <-pr.recvc)
		}
		for _, msg := range msgs {
			if msg.To == pr.id {
				pr.raftGroup.Step(context.TODO(), msg)
			}
		}
	}
}

func (pr *peer) handleReady(rd raft.Ready) {
	pr.ps.saveReadyState(rd)
	for _, state := range rd.ReadStates {
		pr.readStateTable[string(state.RequestCtx)] = state
	}
	if len(rd.ReadStates) > 0 {
		pr.readStateComing <- struct{}{}
	}
	pr.router.sendRaftMsgs(rd.Messages)
	for _, ent := range rd.CommittedEntries {
		pr.process(ent)
	}
	pr.raftGroup.Advance()
}

func (pr *peer) process(ent raftpb.Entry) {
	pr.ps.applyState.ApplyIndex = ent.Index
	pr.ps.writeApplyState(pr.ps.applyState)
	cmd := &raftstorepb.RaftCmdRequest{}
	if err := proto.Unmarshal(ent.Data, cmd); err != nil {
		panic(err)
	}
	if cmd.Request != nil {
		// process common request
		pr.processRequest(cmd.Request)
	} else if cmd.AdminRequest != nil {
		// process admin request
		pr.processAdminRequest(cmd.AdminRequest)
	}
}

func (pr *peer) processRequest(request *raftstorepb.Request) {
	switch request.CmdType {
	case raftstorepb.CmdType_Put:
		logger.Debugf("apply CmdType_Put request: %+v", request.Put)
		modify := storage.PutData(request.Put.Key, request.Put.Value, true)
		pr.ps.engines.WriteKV(modify)
	case raftstorepb.CmdType_Delete:
		logger.Debugf("apply CmdType_Delete request: %+v", request.Delete)
		modify := storage.DeleteData(request.Delete.Key, true)
		pr.ps.engines.WriteKV(modify)
	}
}

func (pr *peer) processAdminRequest(request *raftstorepb.AdminRequest) {
	switch request.CmdType {
	case raftstorepb.AdminCmdType_CompactLog:
		compactLog := request.GetCompactLog()
		applySt := pr.ps.applyState
		if compactLog.CompactIndex >= applySt.TruncatedState.Index {
			applySt.TruncatedState.Index = compactLog.CompactIndex
			applySt.TruncatedState.Term = compactLog.CompactTerm
			pr.ps.writeApplyState(applySt)
			go pr.gcRaftLog(pr.lastCompactedIdx+1, applySt.TruncatedState.Index+1)
			pr.lastCompactedIdx = applySt.TruncatedState.Index
		}
	}
}

func (pr *peer) gcRaftLog(start, end uint64) error {
	logger.Infof("start gc raft log from [%d, %d)", start, end)
	ents, err := pr.ps.Entries(start, end, math.MaxUint64)
	if err != nil {
		return err
	}
	return pr.ps.deleteLogEntries(ents)
}

func (pr *peer) linearizableRead(key []byte) *internal.Callback {
	ctx := buildReadCtx(key)
	cb := internal.NewCallback()
	r := &read{
		ctx: ctx,
		cb:  cb,
	}
	logger.Debugf("receive a read request: %+v", r)
	pr.readc <- r
	if err := pr.raftGroup.ReadIndex(context.TODO(), ctx); err != nil {
		panic(err)
	}
	return cb
}

func (pr *peer) handleReadState() {
	for {
		select {
		case <-pr.readStateComing:
			var r *read
			for len(pr.readc) > 0 {
				r = <-pr.readc
				if _, ok := pr.readStateTable[string(r.ctx)]; ok {
					break
				}
			}
			state := pr.readStateTable[string(r.ctx)]
			delete(pr.readStateTable, string(r.ctx))
			logger.Debugf("ReadState: %+v, ReadRequest: %+v", state, r)
			go pr.readApplied(state, r)
		}
	}
}

func (pr *peer) readApplied(state raft.ReadState, r *read) {
	pr.waitAppliedAdvance(state.Index)
	value, err := pr.ps.engines.ReadKV(r.key())
	if err != nil {
		if err != storage.ErrNotFound {
			panic(err)
		}
	}
	resp := internal.NewGetCmdResponse(value)
	logger.Debugf("%s get response successfully: %+v", string(r.key()), resp.GetResponse())
	r.cb.Done(resp)
}

func (pr *peer) waitAppliedAdvance(index uint64) {
	applied := pr.ps.AppliedIndex()
	if applied >= index {
		return
	}
	donec := make(chan struct{})
	go func() {
		for applied < index {
			time.Sleep(time.Millisecond)
			applied = pr.ps.AppliedIndex()
		}
		donec <- struct{}{}
	}()
	// wait for applied index >= state.Index
	<-donec
	close(donec)
}

func (pr *peer) confState() *raftpb.ConfState {
	c := pr.raftGroup.Status().Config
	return &raftpb.ConfState{
		Voters:         c.Voters[0].Slice(),
		VotersOutgoing: c.Voters[1].Slice(),
		Learners:       quorum.MajorityConfig(c.Learners).Slice(),
		LearnersNext:   quorum.MajorityConfig(c.LearnersNext).Slice(),
		AutoLeave:      c.AutoLeave,
	}
}

func (pr *peer) term() uint64 {
	return pr.raftGroup.Status().Term
}

func (pr *peer) isLeader() bool {
	return pr.raftGroup.Status().Lead == pr.id
}

func (pr *peer) bootstrap() bool {
	return isEmptyConfState(*pr.ps.confState)
}

func buildReadCtx(key []byte) []byte {
	buf := make([]byte, 8)
	ts := time.Now().UnixNano()
	binary.LittleEndian.PutUint64(buf, uint64(ts))
	return append(buf, key...)
}
