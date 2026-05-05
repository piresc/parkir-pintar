import GlassCard from '../ui/GlassCard';
import Button from '../ui/Button';
import StatusBadge from '../ui/StatusBadge';

export default function SpotDetailModal({ spot, onClose, onSelect }) {
  if (!spot) return null;
  return (
    <div className="modal-overlay" onClick={onClose}>
      <GlassCard className="modal-content" onClick={(e) => e.stopPropagation()}>
        <h3>{spot.spot_code}</h3>
        <p>Floor {spot.floor_number}</p>
        <p>Type: {spot.vehicle_type}</p>
        <StatusBadge status={spot.status} />
        {spot.status === 'available' && (
          <Button variant="cta" onClick={() => onSelect(spot)} style={{ marginTop: '1rem' }}>
            Select This Spot
          </Button>
        )}
        <Button variant="ghost" onClick={onClose} style={{ marginTop: '0.5rem' }}>
          Close
        </Button>
      </GlassCard>
    </div>
  );
}
