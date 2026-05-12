import GlassCard from '../ui/GlassCard';
import StatusBadge from '../ui/StatusBadge';
import CountdownTimer from './CountdownTimer';
import { formatDateTime } from '../../utils/formatters';

export default function ReservationCard({ reservation, spotCode }) {
  return (
    <GlassCard className="reservation-card">
      <div className="reservation-header">
        <h3>Your Reservation</h3>
        <StatusBadge status={reservation.status} />
      </div>
      <div className="reservation-detail">
        <span className="label">Spot</span>
        <span className="value spot-code-value">{spotCode || reservation.spot_id}</span>
      </div>
      <div className="reservation-detail">
        <span className="label">Vehicle</span>
        <span className="value">{reservation.vehicle_type}</span>
      </div>
      <div className="reservation-detail">
        <span className="label">Confirmed</span>
        <span className="value">
          {reservation.status === 'waiting_payment' 
            ? 'Pending payment' 
            : formatDateTime(reservation.confirmed_at)}
        </span>
      </div>
      {reservation.status === 'confirmed' && reservation.expires_at && (
        <CountdownTimer target={reservation.expires_at} />
      )}
    </GlassCard>
  );
}
