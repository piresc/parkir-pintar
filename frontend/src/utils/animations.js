export const fadeSlideUp = {
  initial: { opacity: 0, transform: 'translateY(20px)' },
  animate: { opacity: 1, transform: 'translateY(0)' },
  transition: 'all 0.5s cubic-bezier(0.22, 1, 0.36, 1)',
};

export const staggerDelay = (index, baseMs = 30) => ({
  animationDelay: `${index * baseMs}ms`,
});

export function cn(...classes) {
  return classes.filter(Boolean).join(' ');
}
