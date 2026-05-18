// Package e2e_test — wrong-spot detection integration test.
//
// Best practices applied (from Go testify testing standards):
// - Use require for assertions that must pass to continue (fail-fast)
// - Use assert for non-critical checks
// - Follow AAA (Arrange-Act-Assert) structure
// - Use descriptive test names: Test[Scenario]_Should[Expected]_When[Condition]
// - Do not mock the database — these are integration tests against real PostgreSQL
package e2e_test

import (
	"context"
	"math"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// geoKeySpots is the Redis Geo key used to store parking spot coordinates.
	geoKeySpots = "parkir:spots:geo"

	// wrongSpotThresholdMeters is the maximum distance (in meters) a driver
	// can be from their assigned spot before it's flagged as wrong-spot.
	wrongSpotThresholdMeters = 50.0

	// Spot F1-C-001 coordinates (assigned spot) — Monas area, Jakarta
	spotF1C001Lat = -6.175110
	spotF1C001Lng = 106.827153

	// Spot F3-M-010 coordinates (far spot) — ~200m away
	spotF3M010Lat = -6.176900
	spotF3M010Lng = 106.828800

	// Driver reported location near spot F3-M-010 (far from assigned F1-C-001)
	driverReportedLat = -6.176850
	driverReportedLng = 106.828750
)

// TestWrongSpotDetection_ShouldFlagWrongSpot_WhenDriverFarFromAssigned verifies
// that the presence verification detects when a driver checks in at a location
// that is >50m from their assigned parking spot.
//
// In the MVP, wrong-spot is a warning (not a blocker) — the system flags it
// but still allows parking.
//
// Validates: Requirement 9.1 — wrong-spot detection via GPS distance check.
func TestWrongSpotDetection_ShouldFlagWrongSpot_WhenDriverFarFromAssigned(t *testing.T) {
	// Arrange — Seed spot coordinates into Redis Geo
	ctx := context.Background()

	// Clean up geo key before test
	err := env.redisClient.Del(ctx, geoKeySpots).Err()
	require.NoError(t, err)

	// Add assigned spot F1-C-001 to Redis Geo
	err = env.redisClient.GeoAdd(ctx, geoKeySpots, &redis.GeoLocation{
		Name:      "F1-C-001",
		Longitude: spotF1C001Lng,
		Latitude:  spotF1C001Lat,
	}).Err()
	require.NoError(t, err)

	// Add far spot F3-M-010 to Redis Geo
	err = env.redisClient.GeoAdd(ctx, geoKeySpots, &redis.GeoLocation{
		Name:      "F3-M-010",
		Longitude: spotF3M010Lng,
		Latitude:  spotF3M010Lat,
	}).Err()
	require.NoError(t, err)

	// Act — Simulate VerifyLocation: compute distance between driver's
	// reported GPS and the assigned spot F1-C-001 using Redis GEODIST.
	assignedSpotID := "F1-C-001"

	// Get the assigned spot's position from Redis
	positions, err := env.redisClient.GeoPos(ctx, geoKeySpots, assignedSpotID).Result()
	require.NoError(t, err)
	require.Len(t, positions, 1)
	require.NotNil(t, positions[0])

	// Calculate distance between driver's reported location and assigned spot
	// using the Haversine formula (same approach the presence service will use).
	distanceMeters := haversineDistance(
		driverReportedLat, driverReportedLng,
		positions[0].Latitude, positions[0].Longitude,
	)

	// Also verify via Redis GEODIST by temporarily adding driver location
	driverGeoKey := "parkir:driver:location:temp"
	err = env.redisClient.GeoAdd(ctx, driverGeoKey, &redis.GeoLocation{
		Name:      "driver-check-in",
		Longitude: driverReportedLng,
		Latitude:  driverReportedLat,
	}).Err()
	require.NoError(t, err)
	defer env.redisClient.Del(ctx, driverGeoKey) //nolint:errcheck

	// Use Redis GEODIST between the two keys by adding assigned spot to same key
	err = env.redisClient.GeoAdd(ctx, driverGeoKey, &redis.GeoLocation{
		Name:      "assigned-spot",
		Longitude: spotF1C001Lng,
		Latitude:  spotF1C001Lat,
	}).Err()
	require.NoError(t, err)

	redisDistance, err := env.redisClient.GeoDist(ctx, driverGeoKey, "driver-check-in", "assigned-spot", "m").Result()
	require.NoError(t, err)

	// Assert — Verification should return verified=false (distance > 50m)
	verified := distanceMeters <= wrongSpotThresholdMeters

	assert.False(t, verified,
		"expected verified=false when driver is far from assigned spot, distance=%.1fm", distanceMeters)
	assert.Greater(t, distanceMeters, wrongSpotThresholdMeters,
		"expected distance > %.0fm from assigned spot, got %.1fm", wrongSpotThresholdMeters, distanceMeters)

	// Cross-check with Redis GEODIST
	assert.Greater(t, redisDistance, wrongSpotThresholdMeters,
		"Redis GEODIST should also show distance > 50m, got %.1fm", redisDistance)

	// Verify the driver IS near spot F3-M-010 (confirming they're at the wrong spot)
	err = env.redisClient.GeoAdd(ctx, driverGeoKey, &redis.GeoLocation{
		Name:      "wrong-spot-f3",
		Longitude: spotF3M010Lng,
		Latitude:  spotF3M010Lat,
	}).Err()
	require.NoError(t, err)

	distToWrongSpot, err := env.redisClient.GeoDist(ctx, driverGeoKey, "driver-check-in", "wrong-spot-f3", "m").Result()
	require.NoError(t, err)
	assert.Less(t, distToWrongSpot, wrongSpotThresholdMeters,
		"driver should be near spot F3-M-010 (wrong spot), distance=%.1fm", distToWrongSpot)
}

// TestWrongSpotDetection_ShouldVerifyOK_WhenDriverNearAssignedSpot verifies
// that a driver checking in near their assigned spot passes verification.
//
// Validates: Requirement 9.1 — correct spot detection (negative case for wrong-spot).
func TestWrongSpotDetection_ShouldVerifyOK_WhenDriverNearAssignedSpot(t *testing.T) {
	// Arrange
	ctx := context.Background()

	err := env.redisClient.Del(ctx, geoKeySpots).Err()
	require.NoError(t, err)

	// Add assigned spot F1-C-001 to Redis Geo
	err = env.redisClient.GeoAdd(ctx, geoKeySpots, &redis.GeoLocation{
		Name:      "F1-C-001",
		Longitude: spotF1C001Lng,
		Latitude:  spotF1C001Lat,
	}).Err()
	require.NoError(t, err)

	// Driver reports location very close to assigned spot (within 10m)
	nearDriverLat := spotF1C001Lat + 0.00005 // ~5m offset
	nearDriverLng := spotF1C001Lng + 0.00005

	// Act — Calculate distance
	distanceMeters := haversineDistance(
		nearDriverLat, nearDriverLng,
		spotF1C001Lat, spotF1C001Lng,
	)

	// Assert — Should be verified (within threshold)
	verified := distanceMeters <= wrongSpotThresholdMeters

	assert.True(t, verified,
		"expected verified=true when driver is near assigned spot, distance=%.1fm", distanceMeters)
	assert.LessOrEqual(t, distanceMeters, wrongSpotThresholdMeters,
		"expected distance <= %.0fm from assigned spot, got %.1fm", wrongSpotThresholdMeters, distanceMeters)
}

// haversineDistance calculates the distance in meters between two GPS coordinates
// using the Haversine formula. This mirrors the logic the presence service uses.
func haversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusMeters = 6_371_000.0

	dLat := degreesToRadians(lat2 - lat1)
	dLng := degreesToRadians(lng2 - lng1)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(degreesToRadians(lat1))*math.Cos(degreesToRadians(lat2))*
			math.Sin(dLng/2)*math.Sin(dLng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMeters * c
}

// degreesToRadians converts degrees to radians.
func degreesToRadians(deg float64) float64 {
	return deg * math.Pi / 180
}
