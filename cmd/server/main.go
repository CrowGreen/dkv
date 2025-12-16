package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	pb "github.com/CrowGreen/dkv/proto"
	"github.com/CrowGreen/dkv/internal/auth"
	"github.com/CrowGreen/dkv/internal/crypto"
	"github.com/CrowGreen/dkv/internal/server"
	"github.com/CrowGreen/dkv/internal/store"
	"github.com/CrowGreen/dkv/internal/wal"
)

func main() {
	port := flag.Int("port", 5050, "server port")
	dataDir := flag.String("data", "./data", "data directory")
	encKey := flag.String("enc-key", "changeme", "encryption passphrase")
	apiKey := flag.String("api-key", "dev-key-123", "api key for auth")
	tlsCert := flag.String("tls-cert", "", "TLS cert file")
	tlsKey := flag.String("tls-key", "", "TLS key file")
	flag.Parse()

	os.MkdirAll(*dataDir, 0755)

	// init encryption
	enc, err := crypto.NewEncryptor(*encKey)
	if err != nil {
		log.Fatal("crypto init:", err)
	}

	// init store
	s := store.New()

	// restore from snapshot
	snapPath := *dataDir + "/snapshot.enc"
	if err := s.LoadSnapshot(snapPath, enc); err != nil {
		log.Printf("warn: snapshot load: %v", err)
	}

	// init WAL
	walPath := *dataDir + "/wal.log"
	w, err := wal.Open(walPath)
	if err != nil {
		log.Fatal("wal open:", err)
	}
	defer w.Close()

	// replay WAL
	entries, err := w.Replay()
	if err != nil {
		log.Printf("warn: wal replay: %v", err)
	}
	for _, e := range entries {
		switch e.Op {
		case wal.OpPut:
			s.Put(e.Key, e.Value)
		case wal.OpDelete:
			s.Delete(e.Key)
		}
	}
	log.Printf("replayed %d WAL entries, store: %s", len(entries), s.Stats())

	// auth
	keys := auth.NewKeyStore()
	keys.AddKey(*apiKey, "default-client")

	// grpc server
	var opts []grpc.ServerOption
	opts = append(opts,
		grpc.UnaryInterceptor(auth.UnaryInterceptor(keys)),
		grpc.StreamInterceptor(auth.StreamInterceptor(keys)),
	)

	if *tlsCert != "" && *tlsKey != "" {
		creds, err := server.LoadTLSCredentials(server.TLSConfig{
			CertFile: *tlsCert,
			KeyFile:  *tlsKey,
		})
		if err != nil {
			log.Fatal("tls:", err)
		}
		opts = append(opts, grpc.Creds(creds))
		log.Println("TLS enabled")
	}

	grpcServer := grpc.NewServer(opts...)
	kvServer := server.NewKVServer(s, w)
	pb.RegisterKVStoreServer(grpcServer, kvServer)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatal("listen:", err)
	}

	// graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down...")
		grpcServer.GracefulStop()
		s.SaveSnapshot(snapPath, enc)
		log.Println("snapshot saved")
	}()

	log.Printf("dkv server listening on :%d", *port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal("serve:", err)
	}
}
// ensure wal.Sync before snapshot
