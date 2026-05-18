import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

// Custom metrics
const errorRate = new Rate('errors');
const searchDuration = new Trend('search_duration', true);
const reservationDuration = new Trend('reservation_duration', true);

const BASE_URL = __ENV.BASE_URL || 'https://parkir-pintar.piresc.dev';

// Scenarios
export const options = {
  scenarios: {
    // Main load test scenario
    load_test: {
      executor: 'ramping-vus',
      startVUs: 1,
      stages: [
        { duration: '1m', target: 50 },   // ramp up
        { duration: '3m', target: 50 },   // hold steady
        { duration: '30s', target: 0 },   // ramp down
      ],
      exec: 'loadTest',
      tags: { scenario: 'load' },
    },
    // Smoke test scenario for CI (1 VU, 30s)
    smoke_test: {
      executor: 'constant-vus',
      vus: 1,
      duration: '30s',
      exec: 'smokeTest',
      tags: { scenario: 'smoke' },
      startTime: '0s',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<500'],
    errors: ['rate<0.01'],
    search_duration: ['p(95)<500'],
    reservation_duration: ['p(95)<500'],
  },
};

// Allow running a single scenario via environment variable
if (__ENV.SCENARIO === 'smoke') {
  delete options.scenarios.load_test;
} else if (__ENV.SCENARIO === 'load') {
  delete options.scenarios.smoke_test;
}

// NOTE: TEST_JWT_TOKEN env var is required for authenticated endpoints.
const VEHICLE_TYPES = ['car', 'motorcycle'];
const FLOOR_IDS = [1, 2, 3, 4, 5];

function searchAvailability() {
  const vehicleType = VEHICLE_TYPES[randomIntBetween(0, VEHICLE_TYPES.length - 1)];
  const res = http.get(`${BASE_URL}/api/v1/availability?vehicle_type=${vehicleType}`, {
    tags: { name: 'search_availability' },
  });

  searchDuration.add(res.timings.duration);

  const success = check(res, {
    'search availability status 200': (r) => r.status === 200,
    'search availability has body': (r) => r.body && r.body.length > 0,
  });

  errorRate.add(!success);
  return res;
}

function searchFloor() {
  const floorId = FLOOR_IDS[randomIntBetween(0, FLOOR_IDS.length - 1)];
  const res = http.get(`${BASE_URL}/api/v1/floors/${floorId}`, {
    tags: { name: 'search_floor' },
  });

  searchDuration.add(res.timings.duration);

  const success = check(res, {
    'search floor status 200': (r) => r.status === 200,
    'search floor has body': (r) => r.body && r.body.length > 0,
  });

  errorRate.add(!success);
  return res;
}

function createReservation() {
  const payload = JSON.stringify({
    driver_id: `driver-${randomIntBetween(1, 10000)}`,
    vehicle_type: VEHICLE_TYPES[randomIntBetween(0, VEHICLE_TYPES.length - 1)],
    idempotency_key: uuidv4(),
  });

  const params = {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'create_reservation' },
  };

  const res = http.post(`${BASE_URL}/api/v1/reservations`, payload, params);

  reservationDuration.add(res.timings.duration);

  const success = check(res, {
    'reservation created': (r) => r.status === 201 || r.status === 200,
    'reservation has id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.id || body.reservation_id;
      } catch {
        return false;
      }
    },
  });

  errorRate.add(!success);
  return res;
}

function getReservation(reservationId) {
  const res = http.get(`${BASE_URL}/api/v1/reservations/${reservationId}`, {
    tags: { name: 'get_reservation' },
  });

  reservationDuration.add(res.timings.duration);

  const success = check(res, {
    'get reservation status 200': (r) => r.status === 200,
  });

  errorRate.add(!success);
  return res;
}

// Main load test function: 80% search, 20% reservations
export function loadTest() {
  const roll = Math.random();

  if (roll < 0.4) {
    group('Search Availability', () => {
      searchAvailability();
    });
  } else if (roll < 0.8) {
    group('Search Floor', () => {
      searchFloor();
    });
  } else {
    group('Create Reservation', () => {
      const res = createReservation();
      // If reservation was created, fetch it
      try {
        const body = JSON.parse(res.body);
        const id = body.id || body.reservation_id;
        if (id) {
          sleep(0.5);
          getReservation(id);
        }
      } catch {
        // skip get if creation failed
      }
    });
  }

  sleep(randomIntBetween(1, 3));
}

// Smoke test function: exercises all endpoints sequentially
export function smokeTest() {
  group('Smoke - Search Availability', () => {
    searchAvailability();
  });

  sleep(1);

  group('Smoke - Search Floor', () => {
    searchFloor();
  });

  sleep(1);

  group('Smoke - Create and Get Reservation', () => {
    const res = createReservation();
    try {
      const body = JSON.parse(res.body);
      const id = body.id || body.reservation_id;
      if (id) {
        sleep(1);
        getReservation(id);
      }
    } catch {
      // skip if creation failed
    }
  });

  sleep(1);
}
