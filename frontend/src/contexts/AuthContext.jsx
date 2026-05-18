import { createContext, useContext, useState, useCallback, useEffect } from 'react';

const AuthContext = createContext(null);

/**
 * Decode JWT payload without verification (for reading exp claim client-side).
 */
function decodeJWTPayload(token) {
  try {
    const base64 = token.split('.')[1];
    const json = atob(base64.replace(/-/g, '+').replace(/_/g, '/'));
    return JSON.parse(json);
  } catch {
    return null;
  }
}

/**
 * Check if a token is expired by reading the exp claim.
 */
function isTokenExpired(token) {
  const payload = decodeJWTPayload(token);
  if (!payload || !payload.exp) return true;
  // exp is in seconds, Date.now() is in milliseconds
  return Date.now() >= payload.exp * 1000;
}

export function AuthProvider({ children }) {
  const [token, setToken] = useState(() => {
    const stored = localStorage.getItem('pp_token');
    // Clear expired token on load
    if (stored && isTokenExpired(stored)) {
      localStorage.removeItem('pp_token');
      localStorage.removeItem('pp_driver_id');
      return null;
    }
    return stored;
  });
  const [driverId, setDriverId] = useState(() => {
    const stored = localStorage.getItem('pp_token');
    if (stored && isTokenExpired(stored)) return null;
    return localStorage.getItem('pp_driver_id');
  });

  const logout = useCallback(() => {
    localStorage.removeItem('pp_token');
    localStorage.removeItem('pp_driver_id');
    setToken(null);
    setDriverId(null);
  }, []);

  const login = useCallback((newToken, newDriverId) => {
    localStorage.setItem('pp_token', newToken);
    localStorage.setItem('pp_driver_id', newDriverId);
    setToken(newToken);
    setDriverId(newDriverId);
  }, []);

  // Periodically check token expiration
  useEffect(() => {
    if (!token) return;

    const interval = setInterval(() => {
      if (isTokenExpired(token)) {
        logout();
      }
    }, 60_000); // check every minute

    return () => clearInterval(interval);
  }, [token, logout]);

  const isAuthenticated = !!token && !isTokenExpired(token);

  return (
    <AuthContext.Provider value={{ token, driverId, isAuthenticated, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be inside AuthProvider');
  return ctx;
}
