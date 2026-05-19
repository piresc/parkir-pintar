import { createContext, useContext, useState, useCallback, useEffect } from 'react';
import { api } from '../api/client';
import { useAuth } from './AuthContext';

const ReservationContext = createContext(null);

const ACTIVE_STATUSES = ['waiting_payment', 'confirmed', 'checked_in'];

export function ReservationProvider({ children }) {
  const { driverId, isAuthenticated } = useAuth();
  const [currentReservation, setCurrentReservation] = useState(() => {
    const raw = localStorage.getItem('pp_reservation');
    return raw ? JSON.parse(raw) : null;
  });
  const [reservations, setReservations] = useState([]);
  const [loadingReservations, setLoadingReservations] = useState(false);

  const setReservation = useCallback((res) => {
    if (res) {
      localStorage.setItem('pp_reservation', JSON.stringify(res));
    } else {
      localStorage.removeItem('pp_reservation');
    }
    setCurrentReservation(res);
  }, []);

  const clearReservation = useCallback(() => {
    localStorage.removeItem('pp_reservation');
    setCurrentReservation(null);
  }, []);

  const fetchReservations = useCallback(async () => {
    if (!driverId) return [];
    setLoadingReservations(true);
    try {
      const res = await api.getDriverReservations(driverId);
      const list = res.data?.reservations || [];
      setReservations(list);

      // Derive active reservation from API response
      const active = list.find(r => ACTIVE_STATUSES.includes(r.status));
      if (active) {
        setReservation(active);
      } else if (currentReservation && !ACTIVE_STATUSES.includes(currentReservation.status)) {
        // Current reservation is no longer active, clear it
        clearReservation();
      }

      return list;
    } catch (e) {
      console.error('Failed to fetch reservations:', e);
      return [];
    } finally {
      setLoadingReservations(false);
    }
  }, [driverId, currentReservation, setReservation, clearReservation]);

  // Auto-fetch on mount when authenticated
  useEffect(() => {
    if (isAuthenticated && driverId) {
      fetchReservations();
    }
  }, [isAuthenticated, driverId]); // eslint-disable-line react-hooks/exhaustive-deps

  const activeReservation = currentReservation && ACTIVE_STATUSES.includes(currentReservation.status)
    ? currentReservation
    : null;

  const HIDDEN_STATUSES = ['failed', 'expired'];
  const pastReservations = reservations.filter(
    r => !ACTIVE_STATUSES.includes(r.status) && !HIDDEN_STATUSES.includes(r.status)
  );

  return (
    <ReservationContext.Provider
      value={{
        currentReservation,
        activeReservation,
        reservations,
        pastReservations,
        loadingReservations,
        setReservation,
        clearReservation,
        fetchReservations,
      }}
    >
      {children}
    </ReservationContext.Provider>
  );
}

export function useReservation() {
  const ctx = useContext(ReservationContext);
  if (!ctx) throw new Error('useReservation must be inside ReservationProvider');
  return ctx;
}
