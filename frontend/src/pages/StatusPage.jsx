import { useEffect, useState } from 'react';
import { api } from '../api/client';
import HealthStatusCard from '../components/domain/HealthStatusCard';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import ErrorBanner from '../components/ui/ErrorBanner';

export default function StatusPage() {
  const [health, setHealth] = useState(null);
  const [detailed, setDetailed] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  async function load() {
    try {
      setLoading(true);
      setError(null);
      const [h, d] = await Promise.all([api.getHealth(), api.getHealthDetailed()]);
      setHealth(h.data);
      setDetailed(d.data);
    } catch (e) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { load(); }, []);

  const services = detailed?.dependencies
    ? Object.entries(detailed.dependencies).map(([name, info]) => ({ name, ...info }))
    : [];

  return (
    <div className="page status-page">
      <h2>System Health</h2>
      {health && (
        <div className="health-meta">
          <div>Service: {health.service}</div>
          <div>Version: {health.version}</div>
        </div>
      )}
      {loading && <LoadingSpinner />}
      {error && <ErrorBanner message={error} onRetry={load} />}
      <div className="health-grid">
        {services.map((svc) => (
          <HealthStatusCard key={svc.name} name={svc.name} status={svc.status} responseTime={svc.response_time_ms} />
        ))}
      </div>
    </div>
  );
}
