import GlassCard from '../ui/GlassCard';
import StatusBadge from '../ui/StatusBadge';

export default function HealthStatusCard({ name, status, responseTime }) {
  const isHealthy = status === 'UP' || status === 'healthy' || status === 'ok';
  return (
    <GlassCard className="health-card">
      <div className="health-header">
        <span className="health-name">{name}</span>
        <StatusBadge status={isHealthy ? 'available' : 'reserved'} />
      </div>
      {responseTime != null && (
        <div className="health-time">{responseTime}ms</div>
      )}
    </GlassCard>
  );
}
