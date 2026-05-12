import SpotCard from './SpotCard';

export default function FloorGrid({ spots, selectedSpotId, onSelect }) {
  return (
    <div className="floor-grid">
      {spots?.map((spot, i) => (
        <div key={spot.id} className="floor-grid-item" style={{ animationDelay: `${i * 30}ms` }}>
          <SpotCard
            spot={spot}
            isSelected={selectedSpotId === spot.id}
            onClick={() => onSelect(spot)}
          />
        </div>
      ))}
    </div>
  );
}
