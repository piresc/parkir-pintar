export default function GlassCard({ children, className = '', onClick }) {
  return (
    <div
      className={`glass-card ${className}`}
      onClick={onClick}
      style={{ cursor: onClick ? 'pointer' : 'default' }}
    >
      {children}
    </div>
  );
}
