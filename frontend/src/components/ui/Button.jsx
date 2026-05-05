import { cn } from '../../utils/animations';

export default function Button({
  children,
  variant = 'primary',
  className = '',
  disabled = false,
  ...props
}) {
  const variants = {
    primary: 'btn-primary',
    cta: 'btn-cta',
    danger: 'btn-danger',
    ghost: 'btn-ghost',
  };
  return (
    <button
      className={cn('btn', variants[variant] || variants.primary, className)}
      disabled={disabled}
      {...props}
    >
      {children}
    </button>
  );
}
