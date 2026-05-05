import { cn } from '../../utils/animations';

export default function SpotCard({ spot, isSelected, onClick }) {
  const statusColor =
    spot.status === 'available'
      ? 'var(--success)'
      : spot.status === 'reserved'
      ? 'var(--danger)'
      : 'var(--warning)';

  return (
    <div
      className={cn('spot-card', isSelected && 'spot-card-selected')}
      onClick={onClick}
      style={{ borderColor: isSelected ? 'var(--accent-amber)' : statusColor }}
    >
      <div className="spot-code-text" style={{ color: statusColor }}>
        {spot.spot_code}
      </div>
      <div className="spot-type">{spot.vehicle_type}</div>
    </div>
  );
}
