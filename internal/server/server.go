package server

import (
	"context"
	"log"
	"fmt"
	"strings"

	pb "github.com/CrowGreen/dkv/proto"
	"github.com/CrowGreen/dkv/internal/store"
	"github.com/CrowGreen/dkv/internal/wal"
)

type KVServer struct {
	store *store.Store
	wal   *wal.WAL
	pb.UnimplementedKVStoreServer
}

func NewKVServer(s *store.Store, w *wal.WAL) *KVServer {
	return &KVServer{store: s, wal: w}
}

func (s *KVServer) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	if req.Key == "" {
		return &pb.GetResponse{Found: false}, nil
	}
	entry, ok := s.store.Get(req.Key)
	if !ok {
		return &pb.GetResponse{Key: req.Key, Found: false}, nil
	}
	return &pb.GetResponse{
		Key:     req.Key,
		Value:   entry.Value,
		Found:   true,
		Version: entry.Version,
	}, nil
}

func (s *KVServer) Put(ctx context.Context, req *pb.PutRequest) (*pb.PutResponse, error) {
	if req.Key == "" {
		return nil, fmt.Errorf("empty key")
	}

	// write to WAL first
	if s.wal != nil {
		if err := s.wal.Append(wal.Entry{Op: wal.OpPut, Key: req.Key, Value: req.Value}); err != nil {
			log.Printf("wal append failed: %v", err)
			return nil, fmt.Errorf("wal error")
		}
	}

	v := s.store.Put(req.Key, req.Value)
	return &pb.PutResponse{Key: req.Key, Version: v}, nil
}

func (s *KVServer) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	if s.wal != nil {
		s.wal.Append(wal.Entry{Op: wal.OpDelete, Key: req.Key})
	}
	ok := s.store.Delete(req.Key)
	return &pb.DeleteResponse{Deleted: ok}, nil
}

func (s *KVServer) BatchPut(ctx context.Context, req *pb.BatchPutRequest) (*pb.BatchPutResponse, error) {
	count := int32(0)
	for _, e := range req.Entries {
		if e.Key == "" {
			continue
		}
		if s.wal != nil {
			s.wal.Append(wal.Entry{Op: wal.OpPut, Key: e.Key, Value: e.Value})
		}
		s.store.Put(e.Key, e.Value)
		count++
	}
	return &pb.BatchPutResponse{Count: count}, nil
}

func (s *KVServer) Watch(req *pb.WatchRequest, stream pb.KVStore_WatchServer) error {
	ch := s.store.Subscribe(req.Prefix)
	for {
		select {
		case event := <-ch:
			evType := pb.WatchEvent_PUT
			if event.EventType == "DELETE" {
				evType = pb.WatchEvent_DELETE
			}
			if err := stream.Send(&pb.WatchEvent{
				Key:     event.Key,
				Value:   event.Value,
				Type:    evType,
				Version: event.Version,
			}); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return nil
		}
	}
}
// batch put isnt logging properly
