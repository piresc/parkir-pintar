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
  const { activeReservation } = useReservation();

  function getMySpotPath() {
    if (activeReservation?.id) return `/reservation/${activeReservation.id}`;
    return '/my-spot';
  }

  return (
    <nav className="bottom-nav">
      {TABS.map((tab) => {
        const to = tab.path || getMySpotPath();
        const isActive = tab.path
          ? pathname.startsWith(tab.path)
          : pathname.startsWith('/reservation') || pathname === '/my-spot';
        return (
          <Link
            key={tab.label}
            to={to}
            className={cn('bottom-nav-item', isActive && 'active')}
          >
            <span className="bottom-nav-icon">{tab.icon}</span>
            <span className="bottom-nav-label">{tab.label}</span>
          </Link>
        );
      })}
    </nav>
  );
}
