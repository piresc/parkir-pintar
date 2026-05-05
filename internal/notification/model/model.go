// Package model defines domain structs and request types for the notification
// module (stub implementation).
package model

import "time"

// NotificationResult represents the result of a notification send operation.
type NotificationResult struct {
	ID        string    `json:"id"`
	Channel   string    `json:"channel"`
	Status    string    `json:"status"`
	Payload   string    `json:"payload"`
	CreatedAt time.Time `json:"created_at"`
}

// SendPushRequest is the payload for sending a push notification.
type SendPushRequest struct {
	DriverID string            `json:"driver_id"`
	Title    string            `json:"title"`
	Body     string            `json:"body"`
	Data     map[string]string `json:"data,omitzero"`
}

// SendSMSRequest is the payload for sending an SMS notification.
type SendSMSRequest struct {
	PhoneNumber string `json:"phone_number"`
	Message     string `json:"message"`
}

// SendEmailRequest is the payload for sending an email notification.
type SendEmailRequest struct {
	Email   string `json:"email"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}
