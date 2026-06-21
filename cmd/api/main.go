package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	pb "github.com/kamil-koziol/pingo/gen/go"
	"github.com/kamil-koziol/pingo/internal/db"
	"github.com/kamil-koziol/pingo/internal/handler"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	_ "modernc.org/sqlite"
)

func main() {
	grpcAddr := ":50051"
	httpAddr := ":8081"

	// ---- gRPC server ----
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatal(err)
	}

	conn, err := sql.Open("sqlite", "pingo.db")
	if err != nil {
		log.Fatalf("unable to open db: %v", err)
	}

	q := db.New(conn)

	serviceHandler := handler.NewServiceHandler(conn, q)

	grpcServer := grpc.NewServer()
	pb.RegisterServiceServiceServer(grpcServer, serviceHandler)

	go func() {
		log.Println("gRPC listening on", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal(err)
		}
	}()

	// ---- HTTP JSON Gateway ----
	ctx := context.Background()
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	grpcHP := "localhost" + grpcAddr

	if err := pb.RegisterServiceServiceHandlerFromEndpoint(ctx, mux, grpcHP, opts); err != nil {
		log.Fatal(err)
	}

	log.Println("HTTP JSON listening on", httpAddr)
	log.Fatal(http.ListenAndServe(httpAddr, mux))
}
