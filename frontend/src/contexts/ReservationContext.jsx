import { createContext, useContext, useState, useCallback } from 'react';

const ReservationContext = createContext(null);

export function ReservationProvider({ children }) {
  const [currentReservation, setCurrentReservation] = useState(() => {
    const raw = localStorage.getItem('pp_reservation');
    return raw ? JSON.parse(raw) : null;
  });

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

  return (
    <ReservationContext.Provider
      value={{ currentReservation, setReservation, clearReservation }}
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
