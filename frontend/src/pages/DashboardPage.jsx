import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import { useAuth } from '../contexts/AuthContext';
import { useReservation } from '../contexts/ReservationContext';
import GlassCard from '../components/ui/GlassCard';
import Button from '../components/ui/Button';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import ErrorBanner from '../components/ui/ErrorBanner';
import AvailabilityBar from '../components/domain/AvailabilityBar';

export default function DashboardPage() {
  const { driverId } = useAuth();
  const { currentReservation } = useReservation();
  const navigate = useNavigate();
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  async function load() {
    try {
      setLoading(true);
      setError(null);
      const res = await api.getAvailability('');
      setData(res.data);
    } catch (e) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
    const id = setInterval(load, 10000);
    return () => clearInterval(id);
  }, []);

  return (
    <div className="page dashboard-page">
      <h2>Welcome, {driverId}</h2>
      {loading && !data && <LoadingSpinner />}
      {error && <ErrorBanner message={error} onRetry={load} />}
      {data && <AvailabilityBar floors={data.floors} total={data.total} />}
      <div className="dashboard-actions">
        <GlassCard className="action-card" onClick={() => navigate('/reserve?type=car')}>
          <div className="action-icon">🚗</div>
          <div className="action-label">Reserve Car</div>
        </GlassCard>
        <GlassCard className="action-card" onClick={() => navigate('/reserve?type=motorcycle')}>
          <div className="action-icon">🏍️</div>
          <div className="action-label">Reserve Motorcycle</div>
        </GlassCard>
      </div>
      {currentReservation && (
        <GlassCard className="action-card" onClick={() => navigate(`/reservation/${currentReservation.id}`)}>
          <div className="action-label">View Active Reservation</div>
        </GlassCard>
      )}
    </div>
  );
}
