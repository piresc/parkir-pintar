// Package model defines domain structs for the example module with
// validation tags for request binding and database mapping.
package model

import "time"

// Example represents an example domain entity.
type Example struct {
	ID          string    `json:"id" db:"id" validate:"required,uuid"`
	Name        string    `json:"name" db:"name" validate:"required,min=1,max=255"`
	Description string    `json:"description" db:"description"`
	Status      string    `json:"status" db:"status" validate:"required,oneof=active inactive"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// CreateExampleRequest is the payload for creating a new example.
type CreateExampleRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description"`
}

// UpdateExampleRequest is the payload for updating an existing example.
type UpdateExampleRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description"`
	Status      string `json:"status" validate:"required,oneof=active inactive"`
}
