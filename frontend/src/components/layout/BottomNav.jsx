import { Link, useLocation } from 'react-router-dom';
import { cn } from '../../utils/animations';
import { useReservation } from '../../contexts/ReservationContext';

const TABS = [
  { path: '/dashboard', label: 'Home', icon: '🏠' },
  { path: '/floors/1', label: 'Map', icon: '🗺️' },
  { path: null, label: 'My Spot', icon: '🅿️' },
];

export default function BottomNav() {
  const { pathname } = useLocation();
  const { currentReservation } = useReservation();

  function getMySpotPath() {
    if (currentReservation?.id) return `/reservation/${currentReservation.id}`;
    // Fallback: check localStorage for persisted reservation
    try {
      const raw = localStorage.getItem('pp_reservation');
      if (raw) {
        const parsed = JSON.parse(raw);
        if (parsed?.id && !['completed', 'cancelled', 'checked_out'].includes(parsed.status)) {
          return `/reservation/${parsed.id}`;
        }
      }
    } catch {}
    return '/dashboard';
  }

  return (
    <nav className="bottom-nav">
      {TABS.map((tab) => {
        const to = tab.path || getMySpotPath();
        return (
          <Link
            key={tab.label}
            to={to}
            className={cn('bottom-nav-item', pathname.startsWith(tab.path || '/reservation') && 'active')}
          >
            <span className="bottom-nav-icon">{tab.icon}</span>
            <span className="bottom-nav-label">{tab.label}</span>
          </Link>
        );
      })}
    </nav>
  );
}
