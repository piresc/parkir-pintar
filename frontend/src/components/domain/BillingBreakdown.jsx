import GlassCard from '../ui/GlassCard';
import { formatIDR } from '../../utils/formatters';

export default function BillingBreakdown({ bookingFee, parkingFee, overnightFee, total }) {
  const paidBookingFee = bookingFee || 0;
  const remaining = (total || 0) - paidBookingFee;
  const items = [
    { label: 'Parking Fee', value: parkingFee || 0 },
    { label: 'Overnight Fee', value: overnightFee || 0, conditional: true },
  ];

  return (
    <GlassCard className="billing-breakdown">
      <h3>Bill Summary</h3>
      {items.map((item) =>
        item.conditional && item.value === 0 ? null : (
          <div key={item.label} className="billing-row">
            <span>{item.label}</span>
            <span>{formatIDR(item.value)}</span>
          </div>
        )
      )}
      <div className="billing-row" style={{ opacity: 0.6 }}>
        <span>Booking Fee (already paid)</span>
        <span>-{formatIDR(paidBookingFee)}</span>
      </div>
      <div className="billing-row total">
        <span>Amount Due</span>
        <span>{formatIDR(remaining)}</span>
      </div>
    </GlassCard>
  );
}
