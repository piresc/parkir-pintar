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
      const [carRes, motoRes] = await Promise.all([
        api.getAvailability('car'),
        api.getAvailability('motorcycle'),
      ]);
      // Merge car + moto data per floor
      const carFloors = carRes.data?.floors || [];
      const motoFloors = motoRes.data?.floors || [];
      const floors = carFloors.map((cf, i) => ({
        floor_number: cf.floor_number,
        available_car: cf.available_car || 0,
        available_moto: motoFloors[i]?.available_moto || 0,
      }));
      setData({ floors, total: null });
    } catch (e) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
    const id = setInterval(load, 10000);

    // Refetch when tab/page becomes visible again (e.g. navigate back)
    function handleVisibility() {
      if (document.visibilityState === 'visible') load();
    }
    document.addEventListener('visibilitychange', handleVisibility);

    return () => {
      clearInterval(id);
      document.removeEventListener('visibilitychange', handleVisibility);
    };
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

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
