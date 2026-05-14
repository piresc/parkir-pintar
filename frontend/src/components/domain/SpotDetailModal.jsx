import GlassCard from '../ui/GlassCard';
import Button from '../ui/Button';
import StatusBadge from '../ui/StatusBadge';

export default function SpotDetailModal({ spot, onClose, onSelect }) {
  if (!spot) return null;
  return (
    <div className="modal-overlay" role="button" tabIndex={0} onClick={onClose} onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onClose?.(e); } }}>
      <GlassCard className="modal-content" onClick={(e) => e.stopPropagation()}>
        <div className="modal-info">
          <h3>{spot.spot_code}</h3>
          <p>Floor {spot.floor_number}</p>
          <p>Type: {spot.vehicle_type}</p>
          <StatusBadge status={spot.status} />
        </div>
        <div className="modal-actions">
          {spot.status === 'available' && (
            <Button variant="cta" className="modal-btn" onClick={() => onSelect(spot)}>
              Select This Spot
            </Button>
          )}
          <Button variant="ghost" className="modal-btn" onClick={onClose}>
            Close
          </Button>
        </div>
      </GlassCard>
    </div>
  );
}
