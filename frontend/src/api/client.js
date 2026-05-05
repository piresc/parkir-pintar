const API_BASE = import.meta.env.VITE_API_BASE_URL || '';

function getToken() {
  return localStorage.getItem('pp_token');
}

async function apiRequest(method, path, body = null) {
  const headers = {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${getToken() || ''}`,
  };
  const opts = { method, headers };
  if (body) opts.body = JSON.stringify(body);

  const res = await fetch(`${API_BASE}${path}`, opts);
  const data = await res.json().catch(() => ({}));

  if (!res.ok) {
    const err = new Error(data.error || `HTTP ${res.status}`);
    err.status = res.status;
    err.data = data;
    throw err;
  }
  return data;
}

export const api = {
  // Health
  getHealth: () => apiRequest('GET', '/health'),
  getHealthReady: () => apiRequest('GET', '/health/ready'),
  getHealthDetailed: () => apiRequest('GET', '/health/detailed'),

  // Search
  getAvailability: (vehicleType) =>
    apiRequest('GET', `/api/v1/availability?vehicle_type=${vehicleType || ''}`),
  getFloorMap: (floor) => apiRequest('GET', `/api/v1/floors/${floor}`),
  getSpotDetails: (id) => apiRequest('GET', `/api/v1/spots/${id}`),

  // Reservation
  createReservation: (body) => apiRequest('POST', '/api/v1/reservations', body),
  cancelReservation: (id) => apiRequest('DELETE', `/api/v1/reservations/${id}`),
  checkIn: (id) => apiRequest('POST', `/api/v1/reservations/${id}/checkin`),
  checkOut: (id) => apiRequest('POST', `/api/v1/reservations/${id}/checkout`),

  // Presence
  streamLocation: (body) => apiRequest('POST', '/api/v1/presence/stream', body),

  // Payment
  getPaymentStatus: (id) => apiRequest('GET', `/api/v1/payments/${id}/status`),
};
