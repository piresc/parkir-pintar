import Button from './Button';

export default function ErrorBanner({ message, onRetry }) {
  return (
    <div className="error-banner">
      <span>{message}</span>
      {onRetry && (
        <Button variant="ghost" onClick={onRetry}>
          Retry
        </Button>
      )}
    </div>
  );
}
