package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pb "github.com/CrowGreen/dkv/proto"
)

func main() {
	addr := flag.String("addr", "localhost:5050", "server address")
	apiKey := flag.String("key", "dev-key-123", "api key")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("usage: dkv-client [get|put|del|watch] ...")
		os.Exit(1)
	}

	conn, err := grpc.Dial(*addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal("connect:", err)
	}
	defer conn.Close()

	client := pb.NewKVStoreClient(conn)
	ctx := metadata.AppendToOutgoingContext(context.Background(), "x-api-key", *apiKey)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	switch args[0] {
	case "get":
		if len(args) < 2 {
			log.Fatal("usage: get <key>")
		}
		resp, err := client.Get(ctx, &pb.GetRequest{Key: args[1]})
		if err != nil {
			log.Fatal(err)
		}
		if !resp.Found {
			fmt.Printf("(not found) %s\n", args[1])
		} else {
			fmt.Printf("%s = %s (v%d)\n", resp.Key, resp.Value, resp.Version)
		}

	case "put":
		if len(args) < 3 {
			log.Fatal("usage: put <key> <value>")
		}
		resp, err := client.Put(ctx, &pb.PutRequest{
			Key:   args[1],
			Value: []byte(strings.Join(args[2:], " ")),
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("OK %s (v%d)\n", resp.Key, resp.Version)

	case "del":
		if len(args) < 2 {
			log.Fatal("usage: del <key>")
		}
		resp, err := client.Delete(ctx, &pb.DeleteRequest{Key: args[1]})
		if err != nil {
			log.Fatal(err)
		}
		if resp.Deleted {
			fmt.Println("deleted")
		} else {
			fmt.Println("not found")
		}

	case "watch":
		prefix := ""
		if len(args) > 1 {
			prefix = args[1]
		}
		// override timeout for watch
		watchCtx := metadata.AppendToOutgoingContext(context.Background(), "x-api-key", *apiKey)
		stream, err := client.Watch(watchCtx, &pb.WatchRequest{Prefix: prefix})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("watching prefix=%q ...\n", prefix)
		for {
			event, err := stream.Recv()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("[%s] %s = %s (v%d)\n", event.Type, event.Key, event.Value, event.Version)
		}

	default:
		fmt.Printf("unknown command: %s\n", args[0])
		os.Exit(1)
	}
}
// handle ctrl+c cleanly for watch
