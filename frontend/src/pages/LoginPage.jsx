import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import GlassCard from '../components/ui/GlassCard';
import Button from '../components/ui/Button';
import { api } from '../api/client';

export default function LoginPage() {
  const [token, setToken] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();

  async function handleSubmit(e) {
    e.preventDefault();
    const trimmed = token.trim();
    if (!trimmed) return;
    setError('');
    setLoading(true);

    try {
      const res = await api.login(trimmed);
      login(res.data.token, res.data.driver_id);
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
          <label>JWT Token</label>
          <textarea
            value={token}
            onChange={(e) => setToken(e.target.value)}
            placeholder="Paste your JWT token here"
            rows={4}
            style={{
              width: '100%',
              padding: '0.75rem',
              borderRadius: '8px',
              border: '1px solid rgba(255,255,255,0.15)',
              background: 'rgba(255,255,255,0.05)',
              color: 'inherit',
              fontFamily: 'monospace',
              fontSize: '0.85em',
              resize: 'vertical',
            }}
          />

          {error && <p style={{ color: 'var(--error, #ff4444)', fontSize: '0.85em' }}>{error}</p>}
          <Button variant="cta" type="submit" disabled={!token.trim() || loading}>
            {loading ? 'Verifying...' : 'Enter Garage'}
          </Button>
        </form>
      </GlassCard>
    </div>
  );
}
