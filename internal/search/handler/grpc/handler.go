package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pkgconstants "parkir-pintar/internal/constants"
	"parkir-pintar/internal/search"
	"parkir-pintar/internal/search/model"
	searchv1 "parkir-pintar/proto/search/v1"
)

type Handler struct {
	searchv1.UnimplementedSearchServiceServer
	uc search.Usecase
}

func NewHandler(uc search.Usecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) RegisterService(s *grpc.Server) {
	searchv1.RegisterSearchServiceServer(s, h)
}

func (h *Handler) GetAvailability(ctx context.Context, req *searchv1.GetAvailabilityRequest) (*searchv1.AvailabilityResponse, error) {
	if req.GetVehicleType() == "" {
		return nil, status.Error(codes.InvalidArgument, "vehicle_type is required")
	}

	floors, err := h.uc.GetAvailability(ctx, &model.GetAvailabilityRequest{
		VehicleType: req.GetVehicleType(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	protoFloors := make([]*searchv1.FloorAvailability, len(floors))
	var totalAvailable, totalCapacity int32
	for i, f := range floors {
		protoFloors[i] = &searchv1.FloorAvailability{
			FloorNumber:   int32(f.FloorNumber),
			AvailableCar:  int32(f.AvailableCar),
			AvailableMoto: int32(f.AvailableMoto),
			TotalCar:      int32(f.TotalCar),
			TotalMoto:     int32(f.TotalMoto),
		}
		switch req.GetVehicleType() {
		case string(pkgconstants.VehicleTypeCar):
			totalAvailable += int32(f.AvailableCar)
			totalCapacity += int32(f.TotalCar)
		case string(pkgconstants.VehicleTypeMotorcycle):
			totalAvailable += int32(f.AvailableMoto)
			totalCapacity += int32(f.TotalMoto)
		default:
			totalAvailable += int32(f.AvailableCar + f.AvailableMoto)
			totalCapacity += int32(f.TotalCar + f.TotalMoto)
		}
	}

	return &searchv1.AvailabilityResponse{
		Floors: protoFloors,
		Total: &searchv1.AvailabilitySummary{
			TotalAvailable: totalAvailable,
			TotalCapacity:  totalCapacity,
		},
	}, nil
}

func (h *Handler) GetFloorMap(ctx context.Context, req *searchv1.GetFloorMapRequest) (*searchv1.FloorMapResponse, error) {
	if req.GetFloorNumber() < 1 || req.GetFloorNumber() > 5 {
		return nil, status.Error(codes.InvalidArgument, "floor_number must be between 1 and 5")
	}

	spots, err := h.uc.GetFloorMap(ctx, &model.GetFloorMapRequest{
		FloorNumber: int(req.GetFloorNumber()),
	})
	if err != nil {
		return nil, mapError(err)
	}

	protoSpots := make([]*searchv1.SpotInfo, len(spots))
	for i, s := range spots {
		protoSpots[i] = &searchv1.SpotInfo{
			Id:          s.ID,
			SpotCode:    s.SpotCode,
			VehicleType: s.VehicleType,
			Status:      s.Status,
			FloorNumber: int32(s.FloorNumber),
			SpotNumber:  int32(s.SpotNumber),
		}
	}

	return &searchv1.FloorMapResponse{Spots: protoSpots}, nil
}

func (h *Handler) GetSpotDetails(ctx context.Context, req *searchv1.GetSpotDetailsRequest) (*searchv1.SpotDetailsResponse, error) {
	if req.GetSpotId() == "" {
		return nil, status.Error(codes.InvalidArgument, "spot_id is required")
	}

	spot, err := h.uc.GetSpotDetails(ctx, &model.GetSpotDetailsRequest{
		SpotID: req.GetSpotId(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &searchv1.SpotDetailsResponse{
		Id:          spot.ID,
		SpotCode:    spot.SpotCode,
		FloorNumber: int32(spot.FloorNumber),
		SpotNumber:  int32(spot.SpotNumber),
		VehicleType: spot.VehicleType,
		Status:      spot.Status,
	}, nil
}
