import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import { useReservation } from '../contexts/ReservationContext';
import ReservationCard from '../components/domain/ReservationCard';
import LocationSimulator from '../components/domain/LocationSimulator';
import Button from '../components/ui/Button';
import ErrorBanner from '../components/ui/ErrorBanner';
import LoadingSpinner from '../components/ui/LoadingSpinner';

export default function ActiveReservationPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { currentReservation, setReservation, clearReservation } = useReservation();
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(false);

  const reservation = currentReservation;

  async function handleCheckIn() {
    setLoading(true);
    setError(null);
    try {
      const res = await api.checkIn(id);
      setReservation(res.data);
    } catch (e) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }

  async function handleCancel() {
    setLoading(true);
    setError(null);
    try {
      await api.cancelReservation(id);
      clearReservation();
      navigate('/dashboard');
    } catch (e) {
      setError(e.message);
      setLoading(false);
    }
  }

  async function handleLocation(body) {
    setError(null);
    try {
      const res = await api.streamLocation(body);
      if (res.data?.is_geofenced) {
        const checkRes = await api.checkIn(id);
        setReservation(checkRes.data);
      }
    } catch (e) {
      setError(e.message);
    }
  }

  async function handleCheckout() {
    setLoading(true);
    setError(null);
    try {
      const res = await api.checkOut(id);
      setReservation(res.data?.reservation);
      navigate(`/checkout/${id}`);
    } catch (e) {
      setError(e.message);
      setLoading(false);
    }
  }

  if (!reservation) {
    return (
      <div className="page">
        <h2>No active reservation</h2>
        <Button variant="primary" onClick={() => navigate('/dashboard')}>Go Home</Button>
      </div>
    );
  }

  return (
    <div className="page active-reservation-page">
      <ReservationCard reservation={reservation} />
      {error && <ErrorBanner message={error} />}
      {loading && <LoadingSpinner />}
      <div className="action-buttons">
        {reservation.status === 'confirmed' && (
          <>
            <Button variant="primary" onClick={handleCheckIn}>Check In</Button>
            <Button variant="danger" onClick={handleCancel}>Cancel</Button>
          </>
        )}
        {reservation.status === 'checked_in' && (
          <>
            <LocationSimulator reservationId={id} onSend={handleLocation} />
            <Button variant="cta" onClick={handleCheckout}>Check Out</Button>
          </>
        )}
      </div>
    </div>
  );
}
