import GlassCard from '../ui/GlassCard';

export default function AvailabilityBar({ floors, total }) {
  // Compute combined total from floor data (API filters by vehicle_type)
  const combinedTotal = floors?.reduce(
    (sum, f) => sum + (f.available_car || 0) + (f.available_moto || 0),
    0
  ) ?? total?.total_available ?? 0;

  return (
    <div className="availability-bar">
      <div className="availability-total">
        <span className="availability-number">{combinedTotal}</span>
        <span className="availability-label">spots available</span>
      </div>
      <div className="availability-floors">
        {floors?.map((f) => (
          <GlassCard key={f.floor_number} className="availability-floor-card">
            <div className="floor-name">Floor {f.floor_number}</div>
            <div className="floor-counts">
              <span style={{ color: 'var(--accent-cyan)' }}>
                {f.available_car} cars
              </span>
              <span style={{ color: 'var(--accent-amber)' }}>
                {f.available_moto} moto
              </span>
            </div>
          </GlassCard>
        ))}
      </div>
    </div>
  );
}
