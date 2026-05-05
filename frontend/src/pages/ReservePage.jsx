import { useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { useReservation } from '../contexts/ReservationContext';
import { api } from '../api/client';
import { generateIdempotencyKey } from '../utils/formatters';
import GlassCard from '../components/ui/GlassCard';
import Button from '../components/ui/Button';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import ErrorBanner from '../components/ui/ErrorBanner';

export default function ReservePage() {
  const [params] = useSearchParams();
  const navigate = useNavigate();
  const { driverId } = useAuth();
  const { setReservation } = useReservation();
  const [mode, setMode] = useState('system_assigned');
  const [vehicleType, setVehicleType] = useState(params.get('type') || 'car');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  async function handleReserve() {
    setLoading(true);
    setError(null);
    try {
      const res = await api.createReservation({
        driver_id: driverId,
        vehicle_type: vehicleType,
        assignment_mode: mode,
        idempotency_key: generateIdempotencyKey(),
      });
      setReservation(res.data);
      navigate(`/reservation/${res.data.id}`);
    } catch (e) {
      setError(e.message);
      setLoading(false);
    }
  }

  return (
    <div className="page reserve-page">
      <h2>New Reservation</h2>
      <div className="mode-toggle">
        <button className={mode === 'system_assigned' ? 'active' : ''} onClick={() => setMode('system_assigned')}>
          System Assigned
        </button>
        <button className={mode === 'user_selected' ? 'active' : ''} onClick={() => setMode('user_selected')}>
          User Selected
        </button>
      </div>
      <div className="vehicle-toggle">
        <button className={vehicleType === 'car' ? 'active' : ''} onClick={() => setVehicleType('car')}>Car</button>
        <button className={vehicleType === 'motorcycle' ? 'active' : ''} onClick={() => setVehicleType('motorcycle')}>Motorcycle</button>
      </div>
      {mode === 'user_selected' && (
        <GlassCard className="info-card" onClick={() => navigate('/floors/1')}>
          Browse floors to pick your spot →
        </GlassCard>
      )}
      {error && <ErrorBanner message={error} />}
      {loading ? <LoadingSpinner /> : (
        <Button variant="cta" onClick={handleReserve}>
          {mode === 'system_assigned' ? 'Reserve Now' : 'Reserve Selected Spot'}
        </Button>
      )}
    </div>
  );
}
