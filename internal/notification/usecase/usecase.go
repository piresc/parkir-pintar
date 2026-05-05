// Package usecase implements the business logic layer for the notification
// domain module (stub implementation). All methods log the notification
// payload via slog and return a NotificationResult with Status="logged".
package usecase

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"parkir-pintar/internal/notification/model"
)

// Usecase defines the business logic interface for notification operations.
type Usecase interface {
	SendPush(ctx context.Context, req *model.SendPushRequest) (*model.NotificationResult, error)
	SendSMS(ctx context.Context, req *model.SendSMSRequest) (*model.NotificationResult, error)
	SendEmail(ctx context.Context, req *model.SendEmailRequest) (*model.NotificationResult, error)
}

// notificationUsecase is the concrete stub implementation of Usecase.
type notificationUsecase struct{}

// NewUsecase creates a new notification Usecase (stub).
func NewUsecase() Usecase {
	return &notificationUsecase{}
}

// SendPush logs the push notification payload and returns a logged result.
func (uc *notificationUsecase) SendPush(_ context.Context, req *model.SendPushRequest) (*model.NotificationResult, error) {
	payload := marshalPayload(req)
	slog.Info("notification stub: SendPush",
		slog.String("driver_id", req.DriverID),
		slog.String("title", req.Title),
		slog.String("payload", payload))

	return &model.NotificationResult{
		ID:        uuid.New().String(),
		Channel:   "push",
		Status:    "logged",
		Payload:   payload,
		CreatedAt: time.Now(),
	}, nil
}

// SendSMS logs the SMS notification payload and returns a logged result.
func (uc *notificationUsecase) SendSMS(_ context.Context, req *model.SendSMSRequest) (*model.NotificationResult, error) {
	payload := marshalPayload(req)
	slog.Info("notification stub: SendSMS",
		slog.String("phone_number", req.PhoneNumber),
		slog.String("payload", payload))

	return &model.NotificationResult{
		ID:        uuid.New().String(),
		Channel:   "sms",
		Status:    "logged",
		Payload:   payload,
		CreatedAt: time.Now(),
	}, nil
}

// SendEmail logs the email notification payload and returns a logged result.
func (uc *notificationUsecase) SendEmail(_ context.Context, req *model.SendEmailRequest) (*model.NotificationResult, error) {
	payload := marshalPayload(req)
	slog.Info("notification stub: SendEmail",
		slog.String("email", req.Email),
		slog.String("subject", req.Subject),
		slog.String("payload", payload))

	return &model.NotificationResult{
		ID:        uuid.New().String(),
		Channel:   "email",
		Status:    "logged",
		Payload:   payload,
		CreatedAt: time.Now(),
	}, nil
}

// marshalPayload serializes the request to JSON for logging.
func marshalPayload(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}
