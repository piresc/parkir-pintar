import { Link, useLocation } from 'react-router-dom';
import { cn } from '../../utils/animations';

const TABS = [
  { path: '/dashboard', label: 'Home', icon: '🏠' },
  { path: '/floors/1', label: 'Map', icon: '🗺️' },
  { path: '/reservation/current', label: 'My Spot', icon: '🅿️' },
];

export default function BottomNav() {
  const { pathname } = useLocation();
  return (
    <nav className="bottom-nav">
      {TABS.map((tab) => (
        <Link
          key={tab.path}
          to={tab.path}
          className={cn('bottom-nav-item', pathname.startsWith(tab.path.split('/')[1]) && 'active')}
        >
          <span className="bottom-nav-icon">{tab.icon}</span>
          <span className="bottom-nav-label">{tab.label}</span>
        </Link>
      ))}
    </nav>
  );
}
