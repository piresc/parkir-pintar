import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import GlassCard from '../components/ui/GlassCard';
import Button from '../components/ui/Button';
import Input from '../components/ui/Input';

function decodeJWTPayload(token) {
  try {
    const base64 = token.split('.')[1];
    const json = atob(base64.replace(/-/g, '+').replace(/_/g, '/'));
    return JSON.parse(json);
  } catch {
    return null;
  }
}

export default function LoginPage() {
  const [token, setToken] = useState('');
  const [error, setError] = useState('');
  const { login } = useAuth();
  const navigate = useNavigate();

  function handleSubmit(e) {
    e.preventDefault();
    const trimmed = token.trim();
    if (!trimmed) return;
    setError('');

    const payload = decodeJWTPayload(trimmed);
    if (!payload) {
      setError('Invalid JWT token');
      return;
    }

    const driverId = payload.user_id || payload.sub || 'unknown';
    login(trimmed, driverId);
    navigate('/dashboard');
  }

  return (
    <div className="login-page">
      <div className="login-mesh" />
      <GlassCard className="login-card">
        <h1 className="login-title">ParkirPintar</h1>
        <p className="login-subtitle">Smart Parking, Simplified</p>
        <form onSubmit={handleSubmit} className="login-form">
          <label>JWT Token</label>
          <Input
            value={token}
            onChange={(e) => setToken(e.target.value)}
            placeholder="Paste your JWT token"
            style={{ fontSize: '0.8em' }}
          />

          {error && <p style={{ color: 'var(--error, #ff4444)', fontSize: '0.85em' }}>{error}</p>}
          <Button variant="cta" type="submit" disabled={!token.trim()}>
            Enter Garage
          </Button>
        </form>
      </GlassCard>
    </div>
  );
}
