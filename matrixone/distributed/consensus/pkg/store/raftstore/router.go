package raftstore

import (
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/logger"
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/store/raftstore/raft_conn"
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/store/raftstore/raftstorepb"
	"go.etcd.io/etcd/raft/v3/raftpb"
)

// router is the route control center
type router struct {
	addr       string
	raftClient *raft_conn.RaftClient
	raftServer *raft_conn.RaftServer
}

func newRouter(addr string, sender chan<- raftpb.Message) *router {
	r := &router{
		addr:       addr,
		raftServer: raft_conn.NewRaftServer(sender),
		raftClient: raft_conn.NewRaftClient(),
	}
	return r
}

func (r *router) sendRaftMsgs(msgs []raftpb.Message) {
	for _, msg := range msgs {
		addr, ok := peerMap[msg.To]
		if !ok {
			// TODO: handle address does not exist
			continue
		}
		peerMsg := &raftstorepb.RaftMsgReq{
			Message:  &msg,
			FromPeer: msg.From,
			ToPeer:   msg.To,
		}
		logger.Debugf("node %d send msg: %+v", peerMsg.FromPeer, peerMsg.Message.String())
		err := r.raftClient.Send(addr, peerMsg)
		if err != nil {
			// TODO: Handling grpc send failure
			continue
		}
	}
}
