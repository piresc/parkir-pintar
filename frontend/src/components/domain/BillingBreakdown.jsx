import GlassCard from '../ui/GlassCard';
import { formatIDR } from '../../utils/formatters';

export default function BillingBreakdown({ bookingFee, parkingFee, overnightFee, penalty, total }) {
  const items = [
    { label: 'Booking Fee', value: bookingFee || 0 },
    { label: 'Parking Fee', value: parkingFee || 0 },
    { label: 'Overnight Fee', value: overnightFee || 0, conditional: true },
    { label: 'Penalty', value: penalty || 0, conditional: true, danger: true },
  ];

  return (
    <GlassCard className="billing-breakdown">
      <h3>Bill Summary</h3>
      {items.map((item) =>
        item.conditional && item.value === 0 ? null : (
          <div key={item.label} className={`billing-row ${item.danger ? 'danger' : ''}`}>
            <span>{item.label}</span>
            <span>{formatIDR(item.value)}</span>
          </div>
        )
      )}
      <div className="billing-row total">
        <span>Total</span>
        <span>{formatIDR(total || 0)}</span>
      </div>
    </GlassCard>
  );
}
