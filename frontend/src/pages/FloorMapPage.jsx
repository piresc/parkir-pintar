import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import { useReservation } from '../contexts/ReservationContext';
import { useAuth } from '../contexts/AuthContext';
import { generateIdempotencyKey } from '../utils/formatters';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import ErrorBanner from '../components/ui/ErrorBanner';
import FloorGrid from '../components/domain/FloorGrid';
import SpotDetailModal from '../components/domain/SpotDetailModal';
import Button from '../components/ui/Button';

export default function FloorMapPage() {
  const { floor } = useParams();
  const navigate = useNavigate();
  const { driverId } = useAuth();
  const { setReservation } = useReservation();
  const [spots, setSpots] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [selectedSpot, setSelectedSpot] = useState(null);

  async function load() {
    try {
      setLoading(true);
      setError(null);
      const res = await api.getFloorMap(Number(floor));
      setSpots(res.data?.spots || []);
    } catch (e) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { load(); }, [floor]);

  async function reserveSpot(spot) {
    try {
      const res = await api.createReservation({
        driver_id: driverId,
        vehicle_type: spot.vehicle_type,
        assignment_mode: 'user_selected',
        spot_id: spot.id,
        idempotency_key: generateIdempotencyKey(),
      });
      setReservation(res.data);
      navigate(`/reservation/${res.data.id}`);
    } catch (e) {
      setError(e.message);
    }
  }

  return (
    <div className="page floor-map-page">
      <h2>Floor {floor}</h2>
      <div className="floor-tabs">
        {[1,2,3,4,5].map((f) => (
          <Button key={f} variant={Number(floor) === f ? 'primary' : 'ghost'} onClick={() => navigate(`/floors/${f}`)}>
            F{f}
          </Button>
        ))}
      </div>
      {loading && <LoadingSpinner />}
      {error && <ErrorBanner message={error} onRetry={load} />}
      <FloorGrid spots={spots} selectedSpotId={selectedSpot?.id} onSelect={setSelectedSpot} />
      <SpotDetailModal spot={selectedSpot} onClose={() => setSelectedSpot(null)} onSelect={reserveSpot} />
    </div>
  );
}
