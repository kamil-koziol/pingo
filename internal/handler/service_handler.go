package handler

import (
	"context"
	"database/sql"
	"errors"

	pb "github.com/kamil-koziol/pingo/gen/go"
	"github.com/kamil-koziol/pingo/internal/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewServiceHandler(db *sql.DB, q *db.Queries) *ServiceHandler {
	return &ServiceHandler{
		db: db,
		q:  q,
	}
}

type ServiceHandler struct {
	pb.UnimplementedServiceServiceServer
	db *sql.DB
	q  *db.Queries
}

func mapServiceDB(service *db.Service, latestPing *db.Ping) *pb.Service {
	return &pb.Service{
		Id:              service.ID,
		Name:            service.Name,
		Url:             service.Url,
		IsActive:        service.IsActive,
		IntervalSeconds: int32(service.IntervalSeconds),
		ExpectedStatus:  int32(service.ExpectedStatus),
		LatestPing:      mapPingDB(latestPing),
	}
}

func (h *ServiceHandler) GetService(ctx context.Context, r *pb.GetServiceRequest) (*pb.Service, error) {
	service, err := h.q.GetServiceByID(ctx, r.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "service not found")
		}

		return nil, status.Error(codes.Internal, "failed to fetch service")
	}

	latestPings, err := h.q.ListLatestPings(ctx, []int64{service.ID})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch latest ping")
	}

	var latestPing *db.Ping
	if len(latestPings) != 0 {
		latestPing = &latestPings[0].Ping
	}

	return mapServiceDB(&service, latestPing), nil
}

func (h *ServiceHandler) ListServices(ctx context.Context, r *pb.ListServicesRequest) (*pb.ListServicesResponse, error) {
	services, err := h.q.ListServices(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch services")
	}

	servicesIds := make([]int64, len(services))
	for i, service := range services {
		servicesIds[i] = service.ID
	}

	latestPings, err := h.q.ListLatestPings(ctx, servicesIds)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch latest pings")
	}

	latestPingsByService := make(map[int64]*db.Ping)
	for i, ping := range latestPings {
		latestPingsByService[ping.Ping.ServiceID] = &latestPings[i].Ping
	}

	pbServices := make([]*pb.Service, len(services))
	for i := range len(services) {
		pbServices[i] = mapServiceDB(&services[i], latestPingsByService[services[i].ID])
	}

	return &pb.ListServicesResponse{Services: pbServices}, nil
}
