// Package usecase provides property-based tests for spot inventory consistency.
//
// Best practices applied (from Go coding standards KB):
// - Use t.Context() for test context (Go 1.24+)
// - Use table-driven property tests with rapid
// - Test invariants across random reservation lifecycle sequences
package usecase

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"

	"parkir-pintar/internal/reservation/constants"
	"parkir-pintar/internal/reservation/model"
)

// **Validates: Requirements 21.1, 21.2, 21.3, 21.4**
//
// Property 7: Spot Inventory Consistency — spot status matches reservation state;
// total spots = 400 after any operation.

// spotInventory simulates the in-memory spot inventory for property testing.
type spotInventory struct {
	spots        map[string]*model.ParkingSpot // spotID -> spot
	reservations map[string]*model.Reservation // reservationID -> reservation
}

func newSpotInventory() *spotInventory {
	inv := &spotInventory{
		spots:        make(map[string]*model.ParkingSpot),
		reservations: make(map[string]*model.Reservation),
	}
	// Seed 400 spots: 5 floors × (30 car + 50 motorcycle)
	for floor := 1; floor <= 5; floor++ {
		for spot := 1; spot <= 30; spot++ {
			id := fmt.Sprintf("car-f%d-s%d", floor, spot)
			inv.spots[id] = &model.ParkingSpot{
				ID:          id,
				FloorNumber: floor,
				SpotNumber:  spot,
				VehicleType: "car",
				SpotCode:    fmt.Sprintf("F%d-C-%03d", floor, spot),
				Status:      "available",
			}
		}
		for spot := 1; spot <= 50; spot++ {
			id := fmt.Sprintf("moto-f%d-s%d", floor, spot)
			inv.spots[id] = &model.ParkingSpot{
				ID:          id,
				FloorNumber: floor,
				SpotNumber:  spot,
				VehicleType: "motorcycle",
				SpotCode:    fmt.Sprintf("F%d-M-%03d", floor, spot),
				Status:      "available",
			}
		}
	}
	return inv
}

// findAvailableSpot returns the first available spot of the given vehicle type.
func (inv *spotInventory) findAvailableSpot(vehicleType string) *model.ParkingSpot {
	for _, s := range inv.spots {
		if s.VehicleType == vehicleType && s.Status == "available" {
			return s
		}
	}
	return nil
}

// createReservation simulates CreateReservation: spot → "reserved".
func (inv *spotInventory) createReservation(vehicleType string) *model.Reservation {
	spot := inv.findAvailableSpot(vehicleType)
	if spot == nil {
		return nil
	}
	spot.Status = "reserved"
	now := time.Now()
	expiresAt := now.Add(1 * time.Hour)
	r := &model.Reservation{
		ID:             uuid.New().String(),
		DriverID:       uuid.New().String(),
		SpotID:         spot.ID,
		VehicleType:    vehicleType,
		AssignmentMode: string(constants.AssignmentSystemAssigned),
		Status:         string(constants.StatusConfirmed),
		IdempotencyKey: uuid.New().String(),
		ConfirmedAt:    &now,
		ExpiresAt:      &expiresAt,
	}
	inv.reservations[r.ID] = r
	return r
}

// checkIn simulates CheckIn: spot → "occupied".
func (inv *spotInventory) checkIn(reservationID string) bool {
	r, ok := inv.reservations[reservationID]
	if !ok || r.Status != string(constants.StatusConfirmed) {
		return false
	}
	if err := model.ValidateTransition(r.Status, string(constants.StatusCheckedIn)); err != nil {
		return false
	}
	r.Status = string(constants.StatusCheckedIn)
	now := time.Now()
	r.CheckedInAt = &now
	inv.spots[r.SpotID].Status = "occupied"
	return true
}

// checkOut simulates CheckOut: spot → "available".
func (inv *spotInventory) checkOut(reservationID string) bool {
	r, ok := inv.reservations[reservationID]
	if !ok || r.Status != string(constants.StatusCheckedIn) {
		return false
	}
	if err := model.ValidateTransition(r.Status, string(constants.StatusCheckedOut)); err != nil {
		return false
	}
	r.Status = string(constants.StatusCheckedOut)
	now := time.Now()
	r.CheckedOutAt = &now
	inv.spots[r.SpotID].Status = "available"
	return true
}

// cancel simulates CancelReservation: spot → "available".
func (inv *spotInventory) cancel(reservationID string) bool {
	r, ok := inv.reservations[reservationID]
	if !ok || r.Status != string(constants.StatusConfirmed) {
		return false
	}
	if err := model.ValidateTransition(r.Status, string(constants.StatusCancelled)); err != nil {
		return false
	}
	r.Status = string(constants.StatusCancelled)
	now := time.Now()
	r.CancelledAt = &now
	inv.spots[r.SpotID].Status = "available"
	return true
}

// expire simulates ExpireReservation: spot → "available".
func (inv *spotInventory) expire(reservationID string) bool {
	r, ok := inv.reservations[reservationID]
	if !ok || r.Status != string(constants.StatusConfirmed) {
		return false
	}
	if err := model.ValidateTransition(r.Status, string(constants.StatusExpired)); err != nil {
		return false
	}
	r.Status = string(constants.StatusExpired)
	inv.spots[r.SpotID].Status = "available"
	return true
}

// verifyInvariants checks all spot inventory invariants.
func (inv *spotInventory) verifyInvariants(t *testing.T) {
	t.Helper()

	// Invariant 1: Total spots = 400
	require.Equal(t, 400, len(inv.spots), "total spots must remain 400")

	// Count spots by status
	carCount := 0
	motoCount := 0
	for _, s := range inv.spots {
		if s.VehicleType == "car" {
			carCount++
		} else {
			motoCount++
		}
	}
	require.Equal(t, 150, carCount, "car spots must remain 150")
	require.Equal(t, 250, motoCount, "motorcycle spots must remain 250")

	// Invariant 2: Spot status matches reservation state
	// Only check active (non-terminal) reservations, since terminal reservations'
	// spots may have been re-assigned to new reservations.
	activeBySpot := make(map[string]*model.Reservation)
	for _, r := range inv.reservations {
		switch r.Status {
		case string(constants.StatusConfirmed), string(constants.StatusCheckedIn):
			activeBySpot[r.SpotID] = r
		}
	}

	for spotID, r := range activeBySpot {
		spot := inv.spots[spotID]
		switch r.Status {
		case string(constants.StatusConfirmed):
			require.Equal(t, "reserved", spot.Status,
				"spot %s should be reserved for confirmed reservation %s", spot.ID, r.ID)
		case string(constants.StatusCheckedIn):
			require.Equal(t, "occupied", spot.Status,
				"spot %s should be occupied for checked_in reservation %s", spot.ID, r.ID)
		}
	}

	// Verify no spot is reserved/occupied without an active reservation
	for _, spot := range inv.spots {
		if spot.Status == "reserved" || spot.Status == "occupied" {
			_, hasActive := activeBySpot[spot.ID]
			require.True(t, hasActive,
				"spot %s is %s but has no active reservation", spot.ID, spot.Status)
		}
	}
}

func TestProperty_SpotInventoryConsistency(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		inv := newSpotInventory()

		// Generate a random sequence of operations
		numOps := rapid.IntRange(1, 50).Draw(rt, "numOps")
		activeReservations := make([]string, 0)

		for range numOps {
			op := rapid.IntRange(0, 4).Draw(rt, "op")

			switch op {
			case 0: // Create reservation
				vt := rapid.SampledFrom([]string{"car", "motorcycle"}).Draw(rt, "vehicleType")
				r := inv.createReservation(vt)
				if r != nil {
					activeReservations = append(activeReservations, r.ID)
				}
			case 1: // Check in a random confirmed reservation
				if len(activeReservations) > 0 {
					idx := rapid.IntRange(0, len(activeReservations)-1).Draw(rt, "checkinIdx")
					inv.checkIn(activeReservations[idx])
				}
			case 2: // Check out a random checked-in reservation
				if len(activeReservations) > 0 {
					idx := rapid.IntRange(0, len(activeReservations)-1).Draw(rt, "checkoutIdx")
					inv.checkOut(activeReservations[idx])
				}
			case 3: // Cancel a random confirmed reservation
				if len(activeReservations) > 0 {
					idx := rapid.IntRange(0, len(activeReservations)-1).Draw(rt, "cancelIdx")
					inv.cancel(activeReservations[idx])
				}
			case 4: // Expire a random confirmed reservation
				if len(activeReservations) > 0 {
					idx := rapid.IntRange(0, len(activeReservations)-1).Draw(rt, "expireIdx")
					inv.expire(activeReservations[idx])
				}
			}

			// Verify invariants after every operation
			inv.verifyInvariants(t)
		}
	})
}
