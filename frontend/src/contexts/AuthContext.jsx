import { createContext, useContext, useState, useCallback } from 'react';

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [token, setToken] = useState(() => localStorage.getItem('pp_token'));
  const [driverId, setDriverId] = useState(() => localStorage.getItem('pp_driver_id'));

  const login = useCallback((newToken, newDriverId) => {
    localStorage.setItem('pp_token', newToken);
    localStorage.setItem('pp_driver_id', newDriverId);
    setToken(newToken);
    setDriverId(newDriverId);
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem('pp_token');
    localStorage.removeItem('pp_driver_id');
    setToken(null);
    setDriverId(null);
  }, []);

  const isAuthenticated = !!token;

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
