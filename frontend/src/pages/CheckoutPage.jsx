import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import { useReservation } from '../contexts/ReservationContext';
import BillingBreakdown from '../components/domain/BillingBreakdown';
import Button from '../components/ui/Button';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import ErrorBanner from '../components/ui/ErrorBanner';
import StatusBadge from '../components/ui/StatusBadge';

export default function CheckoutPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { currentReservation, clearReservation } = useReservation();
  const [payment, setPayment] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  async function loadPayment() {
    if (!currentReservation?.billing_id) {
      setLoading(false);
      return;
    }
    try {
      const res = await api.getPaymentStatus(currentReservation.billing_id);
      setPayment(res.data);
    } catch (e) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadPayment();
  }, [currentReservation]);

  function handleDone() {
    clearReservation();
    navigate('/dashboard');
  }

  return (
    <div className="page checkout-page">
      <h2>Checkout Complete</h2>
      {loading && <LoadingSpinner />}
      {error && <ErrorBanner message={error} onRetry={loadPayment} />}
      {currentReservation && (
        <BillingBreakdown
          bookingFee={5000}
          parkingFee={currentReservation.parking_fee || 0}
          overnightFee={currentReservation.overnight_fee || 0}
          penalty={currentReservation.penalty_amount || 0}
          total={currentReservation.total_amount || 0}
        />
      )}
      {payment && (
        <div className="payment-status">
          <StatusBadge status={payment.status} />
          <p>Paid at: {payment.paid_at || '-'}</p>
        </div>
      )}
      <Button variant="cta" onClick={handleDone}>Done</Button>
    </div>
  );
}
