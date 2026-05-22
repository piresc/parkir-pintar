const API_BASE = import.meta.env.VITE_API_BASE_URL || '';

function getToken() {
  return localStorage.getItem('pp_token');
}

function clearAuthAndRedirect() {
  // Clear all app data from localStorage
  Object.keys(localStorage)
    .filter(k => k.startsWith('pp_'))
    .forEach(k => localStorage.removeItem(k));
  // Redirect to login page (root route)
  if (window.location.pathname !== '/') {
    window.location.href = '/';
  }
}

async function apiRequest(method, path, body = null) {
  const headers = {
    'Content-Type': 'application/json',
  };

  const token = getToken();
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  const opts = { method, headers };
  if (body) opts.body = JSON.stringify(body);

  const res = await fetch(`${API_BASE}${path}`, opts);
  const data = await res.json().catch(() => ({}));

  if (!res.ok) {
    if (res.status === 401) {
      clearAuthAndRedirect();
    }
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
    apiRequest('GET', `/api/v1/availability?vehicle_type=${vehicleType || 'car'}`),
  getFloorMap: (floor) => apiRequest('GET', `/api/v1/floors/${floor}`),
  getSpotDetails: (id) => apiRequest('GET', `/api/v1/spots/${id}`),

  // Reservation
  createReservation: (body) => apiRequest('POST', '/api/v1/reservations', body),
  getReservation: (id) => apiRequest('GET', `/api/v1/reservations/${id}`),
  getDriverReservations: (driverId, status) => {
    let path = `/api/v1/reservations?driver_id=${driverId}`;
    if (status) path += `&status=${status}`;
    return apiRequest('GET', path);
  },
  cancelReservation: (id) => apiRequest('DELETE', `/api/v1/reservations/${id}`),
  checkIn: (id) => apiRequest('POST', `/api/v1/reservations/${id}/checkin`),
  checkOut: (id) => apiRequest('POST', `/api/v1/reservations/${id}/checkout`),
  confirmReservation: (id) => apiRequest('POST', `/api/v1/reservations/${id}/confirm`),
  completeCheckout: (id) => apiRequest('POST', `/api/v1/reservations/${id}/complete`),

  // Presence
  streamLocation: (body) => apiRequest('POST', '/api/v1/presence/stream', body),

  // Payment
  getPaymentStatus: (id) => apiRequest('GET', `/api/v1/payments/${id}/status`),
};
