package raft_conn

import (
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/logger"
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/store/raftstore/raftstorepb"
	"go.etcd.io/etcd/raft/v3/raftpb"
)

type RaftServer struct {
	msgc chan<- raftpb.Message
}

func NewRaftServer(sender chan<- raftpb.Message) *RaftServer {
	return &RaftServer{
		msgc: sender,
	}
}

func (s *RaftServer) RaftMessage(stream raftstorepb.Message_RaftMessageServer) error {
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		rm := raftMsg(msg)
		logger.Debugf("[grpc] receive msg from %d, msg: %+v", msg.FromPeer, rm.String())
		s.msgc <- rm
	}
}

// raftMsg TODO: rename RaftMsgReq
func raftMsg(msg *raftstorepb.RaftMsgReq) raftpb.Message {
	return *msg.Message
}
