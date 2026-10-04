package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	pb "github.com/kamil-koziol/pingo/gen/go/pingo/v1"
	"github.com/kamil-koziol/pingo/internal/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func NewPingHandler(db *sql.DB, q db.Querier) *PingHandler {
	return &PingHandler{
		db: db,
		q:  q,
	}
}

type PingHandler struct {
	pb.UnimplementedPingServiceServer
	db *sql.DB
	q  db.Querier
}

func mapPingDB(ping *db.Ping) *pb.Ping {
	if ping == nil {
		return nil
	}

	var errorMessage *string = nil
	if ping.ErrorMessage.Valid {
		errorMessage = &ping.ErrorMessage.String
	}

	return &pb.Ping{
		Id:                 ping.ID,
		ServiceId:          ping.ServiceID,
		StatusCode:         int32(ping.StatusCode),
		LatencyMs:          int32(ping.LatencyMs),
		IsUp:               ping.IsUp,
		ErrorMessage:       errorMessage,
		Timestamp:          timestamppb.New(ping.Timestamp),
		ExpectedStatusCode: int32(ping.ExpectedStatusCode),
	}
}

func (h *PingHandler) GetPing(ctx context.Context, r *pb.GetPingRequest) (*pb.Ping, error) {
	ping, err := h.q.GetPing(ctx, h.db, r.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "ping not found")
		}

		return nil, status.Error(codes.Internal, "failed to fetch ping")
	}

	return mapPingDB(ping), nil
}

func (h *PingHandler) ListPings(ctx context.Context, r *pb.ListPingsRequest) (*pb.ListPingsResponse, error) {
	var params db.ListPingsParams
	if r.IsUp != nil {
		params.IsUp = r.IsUp
	}

	if r.ServiceId != nil {
		params.ServiceID = r.ServiceId
	}

	if r.TimestampAfter != nil {
		params.TimestampAfter = r.TimestampAfter.AsTime()
	}

	if r.TimestampBefore != nil {
		params.TimestampBefore = r.TimestampBefore.AsTime()
	}

	pings, err := h.q.ListPings(ctx, h.db, params)
	if err != nil {
		fmt.Println(err)
		return nil, status.Error(codes.Internal, "failed to fetch pings")
	}

	pbPings := make([]*pb.Ping, len(pings))
	for i := range len(pings) {
		pbPings[i] = mapPingDB(pings[i])
	}

	return &pb.ListPingsResponse{Pings: pbPings}, nil
}
