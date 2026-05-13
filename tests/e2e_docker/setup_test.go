//go:build e2e_docker

// Package e2e_docker_test provides Layer 2 E2E tests that boot the full Docker Compose
// stack and send real HTTP requests to the Gateway REST API on localhost:8080.
//
// Best practices applied (from Go testing standards KB):
// - Use descriptive test names: Test[Scenario]_Should[Expected]_When[Condition]
// - Proper AAA (Arrange-Act-Assert) structure
// - Comprehensive coverage of success and error scenarios
// - Use TestMain for shared setup/teardown
package e2e_docker_test

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"testing"

	"github.com/google/uuid"
	"parkir-pintar/tests/testhelpers"
)

// dockerEnvStruct holds shared HTTP client and config for Layer 2 tests.
type dockerEnvStruct struct {
	baseURL    string
	httpClient *http.Client
	jwtToken   string
	driverID   string // seeded test driver UUID
}

// denv is the package-level test environment accessible by all test functions.
var denv *dockerEnvStruct

// projectRoot is the relative path from tests/e2e_docker/ to the project root.
const projectRoot = "../../"

// TestMain manages the Docker Compose lifecycle for all Layer 2 tests.
// It starts the full stack before tests and tears it down after.
func TestMain(m *testing.M) {
	// Start Docker Compose stack
	upCmd := exec.Command("docker", "compose", "up", "-d", "--build", "--wait")
	upCmd.Dir = projectRoot
	upCmd.Stdout = os.Stdout
	upCmd.Stderr = os.Stderr

	log.Println("Starting Docker Compose stack...")
	if err := upCmd.Run(); err != nil {
		log.Fatalf("docker compose up failed: %v", err)
	}

	// Wait for gateway health endpoint
	healthURL := "http://localhost:8082/health"
	log.Printf("Waiting for gateway health at %s ...", healthURL)
	if err := testhelpers.WaitForHealth(healthURL, 120*1e9); err != nil { // 120s
		tearDown()
		log.Fatalf("gateway health check failed: %v", err)
	}
	log.Println("Gateway is healthy.")

	// Seed a test driver into Postgres so reservations have a valid FK.
	driverID := uuid.New().String()
	if err := seedTestDriver(driverID); err != nil {
		tearDown()
		log.Fatalf("seed test driver failed: %v", err)
	}
	log.Printf("Seeded test driver: %s", driverID)

	// Generate JWT token matching JWT_SECRET in Docker Compose .env
	token := testhelpers.GenerateTestJWT(driverID, "driver", "test-jwt-secret")

	// Build authenticated HTTP client
	client := testhelpers.NewAuthenticatedClient(token)

	denv = &dockerEnvStruct{
		baseURL:    "http://localhost:8082",
		httpClient: client,
		jwtToken:   token,
		driverID:   driverID,
	}

	// Run tests
	code := m.Run()

	// Tear down
	tearDown()
	os.Exit(code)
}

// tearDown runs docker compose down -v to clean up all containers and volumes.
func tearDown() {
	log.Println("Tearing down Docker Compose stack...")
	downCmd := exec.Command("docker", "compose", "down", "-v")
	downCmd.Dir = projectRoot
	downCmd.Stdout = os.Stdout
	downCmd.Stderr = os.Stderr

	if err := downCmd.Run(); err != nil {
		// Log warning but don't fail — cleanup is best-effort
		fmt.Fprintf(os.Stderr, "WARNING: docker compose down failed: %v\n", err)
	}
}

// seedTestDriver inserts a single driver row into the Postgres container.
func seedTestDriver(driverID string) error {
	phone := fmt.Sprintf("+628%010d", os.Getpid())
	cmd := exec.Command("docker", "exec", "assessment-postgres", "psql",
		"-U", getEnv("DB_USERNAME", "parkir_user"),
		"-d", getEnv("DB_DATABASE", "parkir_pintar"),
		"-c", fmt.Sprintf(
			`INSERT INTO drivers (id, name, phone, email, vehicle_type, vehicle_plate) VALUES ('%s', 'E2E Test Driver', '%s', 'e2e@test.local', 'car', 'B 1234 E2E') ON CONFLICT (id) DO NOTHING`,
			driverID, phone,
		),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("seed driver: %w (output: %s)", err, string(out))
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
