import { useState, useEffect } from 'react';

import { parseTimestamp } from '../../utils/formatters';

export default function CountdownTimer({ target }) {
  const [remaining, setRemaining] = useState(() => calcRemaining(target));

  useEffect(() => {
    const id = setInterval(() => {
      setRemaining(calcRemaining(target));
    }, 1000);
    return () => clearInterval(id);
  }, [target]);

  function calcRemaining(t) {
    const diff = parseTimestamp(t) - Date.now();
    if (diff <= 0) return { expired: true, text: 'Expired' };
    const m = Math.floor(diff / 60000);
    const s = Math.floor((diff % 60000) / 1000);
    return { expired: false, text: `${m}m ${s}s` };
  }

  return (
    <div className={`countdown-timer ${remaining.expired ? 'expired' : ''}`}>
      <span className="label">Expires in</span>
      <span className="value">{remaining.text}</span>
    </div>
  );
}
