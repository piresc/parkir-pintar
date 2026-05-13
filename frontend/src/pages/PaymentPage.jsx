import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import { useReservation } from '../contexts/ReservationContext';
import { formatIDR } from '../utils/formatters';
import Button from '../components/ui/Button';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import ErrorBanner from '../components/ui/ErrorBanner';

// Reuse the QRIS placeholder from CheckoutPage
function QRISPlaceholder({ amount }) {
  return (
    <div style={{ textAlign: 'center', padding: '1rem' }}>
      <div style={{
        width: 200, height: 200, margin: '0 auto 1rem',
        background: 'white', borderRadius: 12, padding: 12,
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        flexDirection: 'column', gap: 8
      }}>
        <svg width="160" height="160" viewBox="0 0 160 160">
          <rect x="10" y="10" width="40" height="40" rx="4" fill="#1a1a2e" />
          <rect x="15" y="15" width="30" height="30" rx="2" fill="white" />
          <rect x="22" y="22" width="16" height="16" rx="2" fill="#1a1a2e" />
          <rect x="110" y="10" width="40" height="40" rx="4" fill="#1a1a2e" />
          <rect x="115" y="15" width="30" height="30" rx="2" fill="white" />
          <rect x="122" y="22" width="16" height="16" rx="2" fill="#1a1a2e" />
          <rect x="10" y="110" width="40" height="40" rx="4" fill="#1a1a2e" />
          <rect x="15" y="115" width="30" height="30" rx="2" fill="white" />
          <rect x="22" y="122" width="16" height="16" rx="2" fill="#1a1a2e" />
          <rect x="60" y="10" width="8" height="8" fill="#1a1a2e" />
          <rect x="75" y="10" width="8" height="8" fill="#1a1a2e" />
          <rect x="90" y="10" width="8" height="8" fill="#1a1a2e" />
          <rect x="60" y="25" width="8" height="8" fill="#1a1a2e" />
          <rect x="75" y="25" width="8" height="8" fill="#1a1a2e" />
          <rect x="90" y="25" width="8" height="8" fill="#1a1a2e" />
          <rect x="60" y="40" width="8" height="8" fill="#1a1a2e" />
          <rect x="75" y="40" width="8" height="8" fill="#1a1a2e" />
          <rect x="90" y="40" width="8" height="8" fill="#1a1a2e" />
          <rect x="10" y="60" width="8" height="8" fill="#1a1a2e" />
          <rect x="25" y="60" width="8" height="8" fill="#1a1a2e" />
          <rect x="40" y="60" width="8" height="8" fill="#1a1a2e" />
          <rect x="60" y="60" width="8" height="8" fill="#1a1a2e" />
          <rect x="75" y="60" width="8" height="8" fill="#1a1a2e" />
          <rect x="90" y="60" width="8" height="8" fill="#1a1a2e" />
          <rect x="110" y="60" width="8" height="8" fill="#1a1a2e" />
          <rect x="125" y="60" width="8" height="8" fill="#1a1a2e" />
          <rect x="140" y="60" width="8" height="8" fill="#1a1a2e" />
          <rect x="10" y="75" width="8" height="8" fill="#1a1a2e" />
          <rect x="25" y="75" width="8" height="8" fill="#1a1a2e" />
          <rect x="40" y="75" width="8" height="8" fill="#1a1a2e" />
          <rect x="60" y="75" width="8" height="8" fill="#1a1a2e" />
          <rect x="75" y="75" width="8" height="8" fill="#1a1a2e" />
          <rect x="90" y="75" width="8" height="8" fill="#1a1a2e" />
          <rect x="110" y="75" width="8" height="8" fill="#1a1a2e" />
          <rect x="125" y="75" width="8" height="8" fill="#1a1a2e" />
          <rect x="140" y="75" width="8" height="8" fill="#1a1a2e" />
          <rect x="10" y="90" width="8" height="8" fill="#1a1a2e" />
          <rect x="25" y="90" width="8" height="8" fill="#1a1a2e" />
          <rect x="40" y="90" width="8" height="8" fill="#1a1a2e" />
          <rect x="60" y="90" width="8" height="8" fill="#1a1a2e" />
          <rect x="75" y="90" width="8" height="8" fill="#1a1a2e" />
          <rect x="90" y="90" width="8" height="8" fill="#1a1a2e" />
          <rect x="110" y="90" width="8" height="8" fill="#1a1a2e" />
          <rect x="125" y="90" width="8" height="8" fill="#1a1a2e" />
          <rect x="140" y="90" width="8" height="8" fill="#1a1a2e" />
          <rect x="60" y="110" width="8" height="8" fill="#1a1a2e" />
          <rect x="75" y="110" width="8" height="8" fill="#1a1a2e" />
          <rect x="90" y="110" width="8" height="8" fill="#1a1a2e" />
          <rect x="110" y="110" width="8" height="8" fill="#1a1a2e" />
          <rect x="125" y="110" width="8" height="8" fill="#1a1a2e" />
          <rect x="140" y="110" width="8" height="8" fill="#1a1a2e" />
          <rect x="60" y="125" width="8" height="8" fill="#1a1a2e" />
          <rect x="75" y="125" width="8" height="8" fill="#1a1a2e" />
          <rect x="90" y="125" width="8" height="8" fill="#1a1a2e" />
          <rect x="110" y="125" width="8" height="8" fill="#1a1a2e" />
          <rect x="125" y="125" width="8" height="8" fill="#1a1a2e" />
          <rect x="140" y="125" width="8" height="8" fill="#1a1a2e" />
          <rect x="60" y="140" width="8" height="8" fill="#1a1a2e" />
          <rect x="75" y="140" width="8" height="8" fill="#1a1a2e" />
          <rect x="90" y="140" width="8" height="8" fill="#1a1a2e" />
          <rect x="110" y="140" width="8" height="8" fill="#1a1a2e" />
          <rect x="125" y="140" width="8" height="8" fill="#1a1a2e" />
          <rect x="140" y="140" width="8" height="8" fill="#1a1a2e" />
        </svg>
        <span style={{ fontSize: '0.7rem', color: '#666' }}>QRIS Simulation</span>
      </div>
      <p style={{ fontWeight: 600 }}>Scan to pay {formatIDR(amount)}</p>
    </div>
  );
}

export default function PaymentPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { currentReservation, setReservation } = useReservation();
  const [paying, setPaying] = useState(false);
  const [error, setError] = useState(null);
  const [spotCode, setSpotCode] = useState(null);

  useEffect(() => {
    if (currentReservation?.spot_id) {
      api.getSpotDetails(currentReservation.spot_id)
        .then(res => setSpotCode(res.data?.spot_code))
        .catch(() => setSpotCode(null));
    }
  }, [currentReservation]);

  async function handlePay() {
    setPaying(true);
    setError(null);
    try {
      const res = await api.confirmReservation(id);
      setReservation(res.data);
      navigate(`/reservation/${id}`);
    } catch (e) {
      setError(e.message);
      setPaying(false);
    }
  }

  if (!currentReservation) {
    return (
      <div className="page payment-page">
        <h2>Complete Payment</h2>
        <p>No reservation found.</p>
      </div>
    );
  }

  return (
    <div className="page payment-page">
      <div className="payment-content">
        <h2 className="payment-title">Complete Your Reservation</h2>
        <div className="reservation-summary card">
          <p><strong>Spot:</strong> {spotCode || currentReservation.spot_id}</p>
          <p><strong>Vehicle:</strong> {currentReservation.vehicle_type}</p>
        </div>
        <QRISPlaceholder amount={5000} />
        {error && <ErrorBanner message={error} />}
        <Button variant="cta" className="payment-btn" onClick={handlePay} disabled={paying}>
          {paying ? 'Processing...' : 'Pay Booking Fee'}
        </Button>
      </div>
    </div>
  );
}
