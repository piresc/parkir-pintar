import { useState } from 'react';
import Button from './Button';

export default function ErrorBanner({ message, onRetry }) {
  const [dismissed, setDismissed] = useState(false);
  if (dismissed) return null;

  return (
    <div className="error-banner">
      <span>{message}</span>
      <div className="error-actions">
        {onRetry && (
          <Button variant="ghost" onClick={onRetry}>
            Retry
          </Button>
        )}
        <Button variant="ghost" onClick={() => setDismissed(true)}>
          ✕
        </Button>
      </div>
    </div>
  );
}
