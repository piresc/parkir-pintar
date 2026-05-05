import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import GlassCard from '../components/ui/GlassCard';
import Button from '../components/ui/Button';
import Input from '../components/ui/Input';

export default function LoginPage() {
  const [token, setToken] = useState('');
  const [driverId, setDriverId] = useState('');
  const { login } = useAuth();
  const navigate = useNavigate();

  function handleSubmit(e) {
    e.preventDefault();
    if (!token.trim() || !driverId.trim()) return;
    login(token.trim(), driverId.trim());
    navigate('/dashboard');
  }

  return (
    <div className="login-page">
      <div className="login-mesh" />
      <GlassCard className="login-card">
        <h1 className="login-title">ParkirPintar</h1>
        <p className="login-subtitle">Smart Parking, Simplified</p>
        <form onSubmit={handleSubmit} className="login-form">
          <label>Driver ID</label>
          <Input value={driverId} onChange={(e) => setDriverId(e.target.value)} placeholder="e.g. drv-123" />
          <label>JWT Token</label>
          <textarea
            className="input"
            rows={4}
            value={token}
            onChange={(e) => setToken(e.target.value)}
            placeholder="Paste your JWT token..."
          />
          <Button variant="cta" type="submit" disabled={!token || !driverId}>
            Enter Garage
          </Button>
        </form>
      </GlassCard>
    </div>
  );
}
