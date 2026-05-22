import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import GlassCard from '../components/ui/GlassCard';
import Button from '../components/ui/Button';
import { SignJWT } from 'jose';

const DRIVERS = [
  { id: 'driver-1', name: 'Budi Santoso' },
  { id: 'driver-2', name: 'Siti Rahayu' },
  { id: 'driver-3', name: 'Andi Pratama' },
  { id: 'driver-4', name: 'Dewi Lestari' },
];

const JWT_SECRET = import.meta.env.VITE_JWT_SECRET || 'your-super-secret-jwt-key-at-least-32-chars-long';
const JWT_ISSUER = import.meta.env.VITE_JWT_ISSUER || 'parkir-pintar';
const JWT_EXPIRATION_MINUTES = parseInt(import.meta.env.VITE_JWT_EXPIRATION_MINUTES || '60', 10);

async function generateToken(driverId) {
  const secret = new TextEncoder().encode(JWT_SECRET);
  const now = Math.floor(Date.now() / 1000);

  const token = await new SignJWT({ user_id: driverId, role: 'driver' })
    .setProtectedHeader({ alg: 'HS256' })
    .setIssuedAt(now)
    .setIssuer(JWT_ISSUER)
    .setExpirationTime(now + JWT_EXPIRATION_MINUTES * 60)
    .sign(secret);

  return token;
}

export default function LoginPage() {
  const [selected, setSelected] = useState(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();

  async function handleLogin() {
    if (!selected) return;
    setError('');
    setLoading(true);

    try {
      const token = await generateToken(selected.id);
      login(token, selected.id);
      navigate('/dashboard');
    } catch (err) {
      setError(err.message || 'Failed to generate token');
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
        <div className="login-form">
          <label>Select Driver</label>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem', margin: '0.75rem 0' }}>
            {DRIVERS.map((driver) => (
              <button
                key={driver.id}
                type="button"
                onClick={() => setSelected(driver)}
                style={{
                  padding: '0.75rem 1rem',
                  borderRadius: '8px',
                  border: selected?.id === driver.id
                    ? '2px solid var(--accent, #4f8cff)'
                    : '1px solid rgba(255,255,255,0.15)',
                  background: selected?.id === driver.id
                    ? 'rgba(79,140,255,0.15)'
                    : 'rgba(255,255,255,0.05)',
                  color: 'inherit',
                  cursor: 'pointer',
                  textAlign: 'left',
                  transition: 'all 0.2s ease',
                }}
              >
                <span style={{ fontWeight: 600 }}>{driver.name}</span>
                <span style={{ opacity: 0.6, marginLeft: '0.5rem', fontSize: '0.85em' }}>
                  ({driver.id})
                </span>
              </button>
            ))}
          </div>

          {error && <p style={{ color: 'var(--error, #ff4444)', fontSize: '0.85em' }}>{error}</p>}
          <Button variant="cta" type="button" onClick={handleLogin} disabled={!selected || loading}>
            {loading ? 'Signing in...' : 'Enter Garage'}
          </Button>
        </div>
      </GlassCard>
    </div>
  );
}
