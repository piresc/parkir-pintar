import { useState } from 'react';
import Button from '../ui/Button';

const DEMO_COORDS = { lat: -6.2088, lng: 106.8456, accuracy: 5.0 };

export default function LocationSimulator({ reservationId, onSend }) {
  const [coords, setCoords] = useState(DEMO_COORDS);
  const [loading, setLoading] = useState(false);

  async function handleSend() {
    setLoading(true);
    await onSend({
      reservation_id: reservationId,
      latitude: coords.lat,
      longitude: coords.lng,
      accuracy: coords.accuracy,
    });
    setLoading(false);
  }

  return (
    <div className="location-simulator">
      <div className="location-pin">📍</div>
      <div className="location-coords">
        <div>Lat: {coords.lat.toFixed(4)}</div>
        <div>Lng: {coords.lng.toFixed(4)}</div>
        <div>Accuracy: ±{coords.accuracy}m</div>
      </div>
      <Button variant="primary" onClick={handleSend} disabled={loading}>
        {loading ? 'Sending...' : 'Send Location Update'}
      </Button>
    </div>
  );
}
