import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import { useReservation } from '../contexts/ReservationContext';
import ReservationCard from '../components/domain/ReservationCard';
import LocationSimulator from '../components/domain/LocationSimulator';
import Button from '../components/ui/Button';
import ErrorBanner from '../components/ui/ErrorBanner';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import GlassCard from '../components/ui/GlassCard';
import StatusBadge from '../components/ui/StatusBadge';
import { formatDateTime } from '../utils/formatters';

export default function ActiveReservationPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const {
    activeReservation,
    pastReservations,
    loadingReservations,
    setReservation,
    clearReservation,
    fetchReservations,
  } = useReservation();
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(false);
  const [spotCode, setSpotCode] = useState(null);
  const [fetching, setFetching] = useState(false);

  // If we have an ID param but no active reservation matching it, fetch it directly
  useEffect(() => {
    if (id && (!activeReservation || activeReservation.id !== id)) {
      setFetching(true);
      api.getReservation(id)
        .then(res => {
          const data = res.data || res;
          setReservation(data);
        })
        .catch(e => setError(e.message))
        .finally(() => setFetching(false));
    }
  }, [id]); // eslint-disable-line react-hooks/exhaustive-deps

  // Fetch all reservations for history
  useEffect(() => {
    fetchReservations();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // Determine which reservation to display as active
  const reservation = id ? activeReservation : activeReservation;

  useEffect(() => {
    const spotId = reservation?.spot_id;
    if (spotId) {
      api.getSpotDetails(spotId)
        .then(res => setSpotCode(res.data?.spot_code))
        .catch(() => setSpotCode(null));
    }
  }, [reservation?.spot_id]);

  async function handleCheckIn() {
    setLoading(true);
    setError(null);
    try {
      const resId = id || activeReservation?.id;
      const res = await api.checkIn(resId);
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
      const resId = id || activeReservation?.id;
      await api.cancelReservation(resId);
      clearReservation();
      fetchReservations();
      navigate('/my-spot');
    } catch (e) {
      setError(e.message);
      setLoading(false);
    }
  }

  async function handleLocation(body) {
    setError(null);
    try {
      const resId = id || activeReservation?.id;
      const res = await api.streamLocation(body);
      if (res.data?.is_geofenced) {
        const checkRes = await api.checkIn(resId);
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
      const resId = id || activeReservation?.id;
      const res = await api.checkOut(resId);
      setReservation({
        ...res.data?.reservation,
        total_amount: res.data?.total_amount,
        billing_id: res.data?.billing_id,
        payment_id: res.data?.payment_id,
        booking_fee: res.data?.booking_fee,
        parking_fee: res.data?.parking_fee,
        overnight_fee: res.data?.overnight_fee,
        penalty_amount: res.data?.penalty_amount,
      });
      navigate(`/checkout/${resId}`);
    } catch (e) {
      setError(e.message);
      setLoading(false);
    }
  }

  if (fetching || loadingReservations) {
    return (
      <div className="page">
        <LoadingSpinner />
      </div>
    );
  }

  return (
    <div className="page active-reservation-page">
      {/* Active Reservation Section */}
      {reservation ? (
        <>
          <ReservationCard reservation={reservation} spotCode={spotCode} />
          {error && <ErrorBanner message={error} />}
          {loading && <LoadingSpinner />}
          <div className="action-buttons">
            {reservation.status === 'confirmed' && (
              <>
                <Button variant="primary" onClick={handleCheckIn}>Check In</Button>
                <Button variant="danger" onClick={handleCancel}>Cancel</Button>
              </>
            )}
            {reservation.status === 'waiting_payment' && (
              <Button variant="primary" onClick={() => navigate(`/payment/${reservation.id}`)}>
                Complete Payment
              </Button>
            )}
            {reservation.status === 'checked_in' && (
              <>
                <LocationSimulator reservationId={id || reservation.id} onSend={handleLocation} />
                <Button variant="cta" onClick={handleCheckout}>Check Out</Button>
              </>
            )}
          </div>
        </>
      ) : (
        <GlassCard className="no-active-card">
          <div className="no-active-message">
            <span className="no-active-icon">🅿️</span>
            <h3>No Active Reservation</h3>
            <p>You don't have an active parking reservation right now.</p>
            <Button variant="primary" onClick={() => navigate('/dashboard')}>
              Find a Spot
            </Button>
          </div>
        </GlassCard>
      )}

      {/* Reservation History Section */}
      {pastReservations.length > 0 && (
        <div className="reservation-history">
          <h3 className="history-title">History</h3>
          <div className="history-list">
            {pastReservations.map(r => (
              <GlassCard key={r.id} className="history-item">
                <div className="history-item-header">
                  <span className="history-spot">{r.spot_code || r.spot_id}</span>
                  <StatusBadge status={r.status} />
                </div>
                <div className="history-item-details">
                  <span className="history-vehicle">{r.vehicle_type}</span>
                  <span className="history-date">
                    {r.checked_out_at
                      ? formatDateTime(r.checked_out_at)
                      : r.confirmed_at
                        ? formatDateTime(r.confirmed_at)
                        : formatDateTime(r.expires_at)}
                  </span>
                </div>
              </GlassCard>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
