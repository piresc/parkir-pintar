const STATUS_COLORS = {
  confirmed: 'var(--accent-cyan)',
  checked_in: 'var(--success)',
  checked_out: 'var(--text-secondary)',
  expired: 'var(--danger)',
  cancelled: 'var(--danger)',
  available: 'var(--success)',
  reserved: 'var(--danger)',
  success: 'var(--success)',
  failed: 'var(--danger)',
  pending: 'var(--warning)',
};

export default function StatusBadge({ status }) {
  const color = STATUS_COLORS[status?.toLowerCase()] || 'var(--text-secondary)';
  return (
    <span
      className="status-badge"
      style={{ color, borderColor: color }}
    >
      {status?.toUpperCase() || 'UNKNOWN'}
    </span>
  );
}
