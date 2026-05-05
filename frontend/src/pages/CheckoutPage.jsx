import { useNavigate } from 'react-router-dom';
import { useReservation } from '../contexts/ReservationContext';
import BillingBreakdown from '../components/domain/BillingBreakdown';
import Button from '../components/ui/Button';
import StatusBadge from '../components/ui/StatusBadge';

export default function CheckoutPage() {
  const navigate = useNavigate();
  const { currentReservation, clearReservation } = useReservation();

  function handleDone() {
    clearReservation();
    navigate('/dashboard');
  }

  if (!currentReservation) {
    return (
      <div className="page checkout-page">
        <h2>Checkout Complete</h2>
        <p>No checkout data available.</p>
        <Button variant="cta" onClick={handleDone}>Done</Button>
      </div>
    );
  }

  return (
    <div className="page checkout-page">
      <h2>Checkout Complete</h2>
      <BillingBreakdown
        bookingFee={5000}
        parkingFee={currentReservation.parking_fee || 0}
        overnightFee={currentReservation.overnight_fee || 0}
        penalty={currentReservation.penalty_amount || 0}
        total={currentReservation.total_amount || 0}
      />
      <div className="payment-status">
        <StatusBadge status="success" />
        <p>Payment processed successfully via QRIS</p>
      </div>
      <Button variant="cta" onClick={handleDone}>Done</Button>
    </div>
  );
}
