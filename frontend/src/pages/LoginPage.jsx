import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import GlassCard from '../components/ui/GlassCard';
import Button from '../components/ui/Button';
import Input from '../components/ui/Input';
import { generateJWT } from '../utils/jwt';

export default function LoginPage() {
  const [driverId, setDriverId] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const { login } = useAuth();
  const navigate = useNavigate();

  async function handleSubmit(e) {
    e.preventDefault();
    if (!driverId.trim()) return;
    setLoading(true);
    setError('');
    try {
      const token = await generateJWT(driverId.trim());
      login(token, driverId.trim());
      navigate('/dashboard');
    } catch (err) {
      setError(err.message);
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
          <Input value={driverId} onChange={(e) => setDriverId(e.target.value)} placeholder="e.g. driver-1" />
          {error && <p style={{ color: 'var(--error, #ff4444)', fontSize: '0.85em' }}>{error}</p>}
          <Button variant="cta" type="submit" disabled={!driverId.trim() || loading}>
            {loading ? 'Generating...' : 'Enter Garage'}
          </Button>
        </form>
      </GlassCard>
    </div>
  );
}
