package main

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	pb "github.com/kamil-koziol/pingo/gen/go/pingo/v1"
	"github.com/kamil-koziol/pingo/internal/db"
	"github.com/kamil-koziol/pingo/internal/handler"
	"github.com/kamil-koziol/pingo/internal/middleware"
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
	pingHandler := handler.NewPingHandler(conn, q)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.LoggingInterceptor(logger)),
	)
	pb.RegisterServiceServiceServer(grpcServer, serviceHandler)
	pb.RegisterPingServiceServer(grpcServer, pingHandler)

	go func() {
		logger.Info("gRPC listening on", "addr", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("error occured", "err", err)
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

	if err := pb.RegisterPingServiceHandlerFromEndpoint(ctx, mux, grpcHP, opts); err != nil {
		log.Fatal(err)
	}

	logger.Info("HTTP JSON listening on", "addr", httpAddr)
	err = http.ListenAndServe(httpAddr, mux)
	if err != nil {
		logger.Error("error occured", "err", err)
	}
}
