import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import { ReservationProvider } from './contexts/ReservationContext';
import Navbar from './components/layout/Navbar';
import BottomNav from './components/layout/BottomNav';
import LoginPage from './pages/LoginPage';
import DashboardPage from './pages/DashboardPage';
import ReservePage from './pages/ReservePage';
import FloorMapPage from './pages/FloorMapPage';
import ActiveReservationPage from './pages/ActiveReservationPage';
import CheckoutPage from './pages/CheckoutPage';
import StatusPage from './pages/StatusPage';

function ProtectedRoute({ children }) {
  const { isAuthenticated } = useAuth();
  return isAuthenticated ? children : <Navigate to="/" replace />;
}

function AppLayout({ children }) {
  return (
    <div className="app">
      <Navbar />
      <main className="app-main">{children}</main>
      <BottomNav />
    </div>
  );
}

function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<LoginPage />} />
      <Route path="/dashboard" element={<ProtectedRoute><AppLayout><DashboardPage /></AppLayout></ProtectedRoute>} />
      <Route path="/reserve" element={<ProtectedRoute><AppLayout><ReservePage /></AppLayout></ProtectedRoute>} />
      <Route path="/floors/:floor" element={<ProtectedRoute><AppLayout><FloorMapPage /></AppLayout></ProtectedRoute>} />
      <Route path="/reservation/:id" element={<ProtectedRoute><AppLayout><ActiveReservationPage /></AppLayout></ProtectedRoute>} />
      <Route path="/checkout/:id" element={<ProtectedRoute><AppLayout><CheckoutPage /></AppLayout></ProtectedRoute>} />
      <Route path="/status" element={<ProtectedRoute><AppLayout><StatusPage /></AppLayout></ProtectedRoute>} />
    </Routes>
  );
}

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <ReservationProvider>
          <AppRoutes />
        </ReservationProvider>
      </AuthProvider>
    </BrowserRouter>
  );
}
