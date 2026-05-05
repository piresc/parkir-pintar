import { useAuth } from '../../contexts/AuthContext';
import Button from '../ui/Button';

export default function Navbar() {
  const { driverId, logout } = useAuth();
  return (
    <nav className="navbar">
      <div className="navbar-brand">ParkirPintar</div>
      <div className="navbar-user">
        <span>{driverId || 'Guest'}</span>
        <Button variant="ghost" onClick={logout}>Logout</Button>
      </div>
    </nav>
  );
}
