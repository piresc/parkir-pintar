package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"parkir-pintar/internal/search/model"
	"parkir-pintar/internal/search/repository"
	searchv1 "parkir-pintar/proto/search/v1"
)

// mockUsecase is a testify mock for the search usecase.Usecase interface.
type mockUsecase struct {
	mock.Mock
}

func (m *mockUsecase) GetAvailability(ctx context.Context, req *model.GetAvailabilityRequest) ([]model.FloorAvailability, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.FloorAvailability), args.Error(1)
}

func (m *mockUsecase) GetFloorMap(ctx context.Context, req *model.GetFloorMapRequest) ([]model.SpotDetails, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.SpotDetails), args.Error(1)
}

func (m *mockUsecase) GetSpotDetails(ctx context.Context, req *model.GetSpotDetailsRequest) (*model.SpotDetails, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.SpotDetails), args.Error(1)
}

func TestGetAvailability(t *testing.T) {
	tests := []struct {
		name       string
		req        *searchv1.GetAvailabilityRequest
		mockResult []model.FloorAvailability
		mockErr    error
		wantCode   codes.Code
	}{
		{
			name: "happy path car",
			req: &searchv1.GetAvailabilityRequest{
				VehicleType: "car",
			},
			mockResult: []model.FloorAvailability{
				{FloorNumber: 1, AvailableCar: 10, TotalCar: 20, AvailableMoto: 5, TotalMoto: 10},
				{FloorNumber: 2, AvailableCar: 8, TotalCar: 20, AvailableMoto: 3, TotalMoto: 10},
			},
			wantCode: codes.OK,
		},
		{
			name: "missing vehicle_type",
			req: &searchv1.GetAvailabilityRequest{
				VehicleType: "",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "usecase returns error",
			req: &searchv1.GetAvailabilityRequest{
				VehicleType: "car",
			},
			mockResult: nil,
			mockErr:    errors.New("db connection error"),
			wantCode:   codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &mockUsecase{}
			h := NewHandler(uc)

			if tt.req.GetVehicleType() != "" {
				uc.On("GetAvailability", mock.Anything, mock.Anything).Return(tt.mockResult, tt.mockErr)
			}

			resp, err := h.GetAvailability(t.Context(), tt.req)

			if tt.wantCode == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Len(t, resp.GetFloors(), len(tt.mockResult))
				assert.NotNil(t, resp.GetTotal())
				// For "car" type, total should sum AvailableCar
				assert.Equal(t, int32(18), resp.GetTotal().GetTotalAvailable())
				assert.Equal(t, int32(40), resp.GetTotal().GetTotalCapacity())
			} else {
				assert.Nil(t, resp)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
			}

			uc.AssertExpectations(t)
		})
	}
}

func TestGetFloorMap(t *testing.T) {
	tests := []struct {
		name       string
		req        *searchv1.GetFloorMapRequest
		mockResult []model.SpotDetails
		mockErr    error
		wantCode   codes.Code
	}{
		{
			name: "happy path",
			req: &searchv1.GetFloorMapRequest{
				FloorNumber: 1,
			},
			mockResult: []model.SpotDetails{
				{ID: "spot-1", SpotCode: "1A01", FloorNumber: 1, SpotNumber: 1, VehicleType: "car", Status: "available"},
				{ID: "spot-2", SpotCode: "1A02", FloorNumber: 1, SpotNumber: 2, VehicleType: "car", Status: "occupied"},
			},
			wantCode: codes.OK,
		},
		{
			name: "invalid floor_number zero",
			req: &searchv1.GetFloorMapRequest{
				FloorNumber: 0,
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "invalid floor_number too high",
			req: &searchv1.GetFloorMapRequest{
				FloorNumber: 6,
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "invalid floor_number negative",
			req: &searchv1.GetFloorMapRequest{
				FloorNumber: -1,
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "usecase returns error",
			req: &searchv1.GetFloorMapRequest{
				FloorNumber: 2,
			},
			mockResult: nil,
			mockErr:    repository.ErrNotFound,
			wantCode:   codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &mockUsecase{}
			h := NewHandler(uc)

			if tt.req.GetFloorNumber() >= 1 && tt.req.GetFloorNumber() <= 5 {
				uc.On("GetFloorMap", mock.Anything, mock.Anything).Return(tt.mockResult, tt.mockErr)
			}

			resp, err := h.GetFloorMap(t.Context(), tt.req)

			if tt.wantCode == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Len(t, resp.GetSpots(), len(tt.mockResult))
				assert.Equal(t, "spot-1", resp.GetSpots()[0].GetId())
				assert.Equal(t, "1A01", resp.GetSpots()[0].GetSpotCode())
			} else {
				assert.Nil(t, resp)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
			}

			uc.AssertExpectations(t)
		})
	}
}

func TestGetSpotDetails(t *testing.T) {
	tests := []struct {
		name       string
		req        *searchv1.GetSpotDetailsRequest
		mockResult *model.SpotDetails
		mockErr    error
		wantCode   codes.Code
	}{
		{
			name: "happy path",
			req: &searchv1.GetSpotDetailsRequest{
				SpotId: "spot-123",
			},
			mockResult: &model.SpotDetails{
				ID:          "spot-123",
				SpotCode:    "2B05",
				FloorNumber: 2,
				SpotNumber:  5,
				VehicleType: "motorcycle",
				Status:      "available",
			},
			wantCode: codes.OK,
		},
		{
			name: "missing spot_id",
			req: &searchv1.GetSpotDetailsRequest{
				SpotId: "",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "usecase returns not found",
			req: &searchv1.GetSpotDetailsRequest{
				SpotId: "spot-999",
			},
			mockResult: nil,
			mockErr:    repository.ErrNotFound,
			wantCode:   codes.NotFound,
		},
		{
			name: "usecase returns internal error",
			req: &searchv1.GetSpotDetailsRequest{
				SpotId: "spot-123",
			},
			mockResult: nil,
			mockErr:    errors.New("unexpected error"),
			wantCode:   codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &mockUsecase{}
			h := NewHandler(uc)

			if tt.req.GetSpotId() != "" {
				uc.On("GetSpotDetails", mock.Anything, mock.Anything).Return(tt.mockResult, tt.mockErr)
			}

			resp, err := h.GetSpotDetails(t.Context(), tt.req)

			if tt.wantCode == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.mockResult.ID, resp.GetId())
				assert.Equal(t, tt.mockResult.SpotCode, resp.GetSpotCode())
				assert.Equal(t, int32(tt.mockResult.FloorNumber), resp.GetFloorNumber())
				assert.Equal(t, tt.mockResult.VehicleType, resp.GetVehicleType())
				assert.Equal(t, tt.mockResult.Status, resp.GetStatus())
			} else {
				assert.Nil(t, resp)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
			}

			uc.AssertExpectations(t)
		})
	}
}
