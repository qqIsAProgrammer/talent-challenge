package raftstore

import (
	"errors"
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/logger"
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/store/raftstore/internal"
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/store/raftstore/raftstorepb"
	"github.com/matrixorigin/talent-challenge/matrixbase/distributed/pkg/store/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"net"
	"time"
)

var (
	ErrLostReadResponse = errors.New("it lost read response, retry read")
)

var peerMap = map[uint64]string{
	1: "127.0.0.1:6060", // rpc port
	2: "127.0.0.1:6061",
	3: "127.0.0.1:6062",
}

type RaftStore struct {
	pr *peer
}

func NewRaftStore(storeId uint64, dataPath string) *RaftStore {
	logger.SetLevel(logger.LogLevelInfo)
	rs := &RaftStore{pr: newPeer(storeId, dataPath)}
	go rs.serveGrpc(storeId)
	return rs
}

func (rs *RaftStore) serveGrpc(id uint64) {
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

	raftstorepb.RegisterMessageServer(srv, rs.pr.router.raftServer)
	srv.Serve(l)
}

func (rs *RaftStore) Set(key []byte, value []byte) error {
	header := &raftstorepb.RaftRequestHeader{Term: rs.pr.term()}
	req := &raftstorepb.Request{
		CmdType: raftstorepb.CmdType_Put,
		Put: &raftstorepb.PutRequest{
			Key:   key,
			Value: value,
		},
	}
	cmd := internal.NewRaftCmdRequest(header, req)
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
	header := &raftstorepb.RaftRequestHeader{Term: rs.pr.term()}
	req := &raftstorepb.Request{
		CmdType: raftstorepb.CmdType_Delete,
		Delete: &raftstorepb.DeleteRequest{
			Key: key,
		},
	}
	cmd := internal.NewRaftCmdRequest(header, req)
	return rs.pr.propose(cmd)
}
