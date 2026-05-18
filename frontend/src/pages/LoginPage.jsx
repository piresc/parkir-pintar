import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import GlassCard from '../components/ui/GlassCard';
import Button from '../components/ui/Button';
import Input from '../components/ui/Input';
import { api } from '../api/client';

export default function LoginPage() {
  const [driverId, setDriverId] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();

  async function handleSubmit(e) {
    e.preventDefault();
    const trimmed = driverId.trim();
    if (!trimmed) return;
    setError('');
    setLoading(true);

    try {
      const data = await api.login(trimmed);
      login(data.token, trimmed);
      navigate('/dashboard');
    } catch (err) {
      setError(err.message || 'Login failed');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="login-page">
      <div className="login-mesh" />
      <GlassCard className="login-card">
        <h1 className="login-title">ParkirPintar</h1>
        <p className="login-subtitle">Smart Parking, Simplified</p>
        <form onSubmit={handleSubmit} className="login-form">
          <label>Driver ID</label>
          <Input
            value={driverId}
            onChange={(e) => setDriverId(e.target.value)}
            placeholder="Enter your driver ID"
          />

          {error && <p style={{ color: 'var(--error, #ff4444)', fontSize: '0.85em' }}>{error}</p>}
          <Button variant="cta" type="submit" disabled={!driverId.trim() || loading}>
            {loading ? 'Signing in...' : 'Enter Garage'}
          </Button>
        </form>
      </GlassCard>
    </div>
  );
}
