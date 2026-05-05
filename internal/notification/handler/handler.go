// Package handler provides gRPC handlers for the notification domain module.
// Handlers map proto requests to usecase calls and return proto responses.
package handler

import (
	"context"

	"google.golang.org/grpc"

	"parkir-pintar/internal/notification/model"
	"parkir-pintar/internal/notification/usecase"
	notificationv1 "parkir-pintar/proto/notification/v1"
)

// Handler implements the notificationv1.NotificationServiceServer gRPC interface.
type Handler struct {
	notificationv1.UnimplementedNotificationServiceServer
	uc usecase.Usecase
}

// NewHandler creates a new notification gRPC Handler with the given usecase.
func NewHandler(uc usecase.Usecase) *Handler {
	return &Handler{uc: uc}
}

// RegisterService registers this handler with the given gRPC server.
func (h *Handler) RegisterService(s *grpc.Server) {
	notificationv1.RegisterNotificationServiceServer(s, h)
}

// SendPush maps the proto request to the usecase and returns a proto response.
func (h *Handler) SendPush(ctx context.Context, req *notificationv1.SendPushRequest) (*notificationv1.NotificationResponse, error) {
	result, err := h.uc.SendPush(ctx, &model.SendPushRequest{
		DriverID: req.GetDriverId(),
		Title:    req.GetTitle(),
		Body:     req.GetBody(),
		Data:     req.GetData(),
	})
	if err != nil {
		return nil, err
	}
	return toProto(result), nil
}

// SendSMS maps the proto request to the usecase and returns a proto response.
func (h *Handler) SendSMS(ctx context.Context, req *notificationv1.SendSMSRequest) (*notificationv1.NotificationResponse, error) {
	result, err := h.uc.SendSMS(ctx, &model.SendSMSRequest{
		PhoneNumber: req.GetPhoneNumber(),
		Message:     req.GetMessage(),
	})
	if err != nil {
		return nil, err
	}
	return toProto(result), nil
}

// SendEmail maps the proto request to the usecase and returns a proto response.
func (h *Handler) SendEmail(ctx context.Context, req *notificationv1.SendEmailRequest) (*notificationv1.NotificationResponse, error) {
	result, err := h.uc.SendEmail(ctx, &model.SendEmailRequest{
		Email:   req.GetEmail(),
		Subject: req.GetSubject(),
		Body:    req.GetBody(),
	})
	if err != nil {
		return nil, err
	}
	return toProto(result), nil
}

// toProto converts a domain NotificationResult to a proto NotificationResponse.
func toProto(r *model.NotificationResult) *notificationv1.NotificationResponse {
	if r == nil {
		return nil
	}
	return &notificationv1.NotificationResponse{
		Id:      r.ID,
		Channel: r.Channel,
		Status:  r.Status,
		Payload: r.Payload,
	}
}
