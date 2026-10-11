import React, { useId } from 'react';

interface FireIconProps {
  width?: number | string;
  height?: number | string;
  className?: string;
}

// Three soft flames, with a cluster of smaller amber ones at their base,
// resting on a pile of twigs. A peach flame leaning left sits behind a
// terracotta one leaning right, with a deeper rust gradient flame tucked behind
// on the left; their tips step high / mid / low so the silhouette reads as a
// fire rather than a single drop. Every fill is opaque so the mark looks the
// same on the cream and the dark background; only the ground shadow changes
// with the theme. The wood is darker than the flames so the fire sits on it.
//
// The favicon (src/app/icon.svg) is a static copy of this artwork; change both
// together.

// The terracotta flame's outline, reused (scaled, sometimes mirrored) for every
// flame in the core cluster.
const FLAME_PATH =
  'M36 13C42 24 50 33 50 43C50 52 43 58 35 58C27 58 22 52 22 45C22 35 31 28 36 13Z';

// The core cluster: smaller amber-to-burnt flames layered over the seams where
// the three large flames overlap, listed back to front. Each sits on the logs
// (base anchored at y=54, tucked behind them) at horizontal position `x`; a
// negative `sx` mirrors it. `sy` runs a little above `sx` so they rise without
// looking stretched.
const CORE_FLAMES = [
  { fill: '#EC9B5F', x: 40, sx: -0.44, sy: 0.52 },
  { fill: '#E8925A', x: 20, sx: 0.58, sy: 0.66 },
  { fill: '#F0AE68', x: 31, sx: -0.62, sy: 0.72 },
  { fill: '#DD844F', x: 12, sx: 0.3, sy: 0.36 },
  { fill: '#D4733F', x: 25.5, sx: 0.36, sy: 0.42 },
  { fill: '#F5C07E', x: 36, sx: -0.26, sy: 0.32 },
  { fill: '#C9663A', x: 18.5, sx: 0.22, sy: 0.27 },
  { fill: '#FAD49A', x: 29, sx: -0.2, sy: 0.26 },
] as const;

// The woodpile: twigs stacked at different angles, listed back to front. Each
// is a run of straight segments that kinks along its length like a lightning
// bolt, tapers toward one end and is snapped off at a slant; a few short side
// shoots poke out to a point. Flat colour, no grain lines. A low filler in the
// darkest wood colour sits behind them so the centre of the pile reads as
// solid.
const WOOD_FILLER =
  'M14 54.5C22 52.4 42 52 51 53.4C51.6 56.4 14.4 57.6 14 54.5Z';
const STICKS = [
  { fill: '#4F3428', d: 'M6.7 57.7L16.1 56.4L23.1 56.6L33.3 54.3L42.3 53.4L50.2 50.9L57.5 49.8L55.9 48.3L49.8 48.5L41.7 50.4L32.7 50.9L22.9 53.6L15.9 52.8L7.2 54.3Z' },
  { fill: '#4F3428', d: 'M16.5 53.9L14.3 51.5L12 50L13.8 52L15.5 54.9Z' },
  { fill: '#5A3B2E', d: 'M7.2 51.4L17.9 52.9L25.9 52.3L34.8 54.5L43.8 54.7L50.7 56.3L55.4 57L57.1 55.9L51.3 54.1L44.2 51.9L35.2 51.3L26.1 49.5L18.1 49.7L8.7 48.6Z' },
  { fill: '#62412F', d: 'M12.6 59.8L21.1 58.8L30.1 59.2L39.3 56.9L49 54.8L47.4 53.4L38.7 54.5L29.9 56.6L20.9 55.8L13.3 57Z' },
  { fill: '#62412F', d: 'M29.6 58.4L31.5 60.1L33.5 61L31.9 59.7L30.4 57.6Z' },
  { fill: '#74503F', d: 'M9.6 53.3L18.9 55.1L26.8 54.6L35.7 57.3L44.7 58.1L51.8 59.9L59.4 59.7L58 58.1L52.2 57.5L45.3 55.1L36.3 54.1L27.2 51.6L19.1 51.7L11.3 50.3Z' },
  { fill: '#74503F', d: 'M22.6 53.6L24 51.1L24.5 48.6L23.4 50.7L21.4 53Z' },
  { fill: '#74503F', d: 'M46.6 57.6L49.5 59.5L51.5 61.4L49.9 59L47.4 56.8Z' },
  { fill: '#6B4736', d: 'M19.1 61.5L26.6 60.3L33.1 60.7L40.6 58.1L38.8 57.2L32.9 58.7L26.4 57.9L19.8 59.3Z' },
  { fill: '#74503F', d: 'M43.2 58.1L46.6 54.6L48 52L51.9 49.4L50.8 49.2L47.2 51.2L45.4 53.6L42.7 56Z' },
] as const;

const FireIcon: React.FC<FireIconProps> = ({
  width = 36,
  height = 36,
  className = '',
}) => {
  // Several logos can share a page; per-instance ids keep their gradient and
  // filter definitions from colliding.
  const id = useId();
  const gradientId = `${id}-flame`;
  const glowId = `${id}-glow`;
  const shadowBlurId = `${id}-shadow`;

  return (
    <svg
      width={width}
      height={height}
      viewBox="0 0 64 64"
      className={className}
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
    >
      <defs>
        <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="#E58A68" />
          <stop offset="100%" stopColor="#B9583F" />
        </linearGradient>
        <radialGradient id={glowId} cx="32" cy="38" r="30" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stopColor="#F7A66B" stopOpacity={0.22} />
          <stop offset="60%" stopColor="#F7A66B" stopOpacity={0.08} />
          <stop offset="100%" stopColor="#F7A66B" stopOpacity={0} />
        </radialGradient>
        <filter id={shadowBlurId} x="-30%" y="-150%" width="160%" height="400%">
          <feGaussianBlur stdDeviation="1.8" />
        </filter>
      </defs>
      {/* A fuzzy shadow cast along the ground: roughly the fire's silhouette
          lying on its side, wide under the woodpile and trailing off to a tip
          on the right. The one place the mark is theme-aware: a brown shadow
          vanishes on the dark background, so dark mode swaps it for a deep
          black. */}
      <path
        d="M7 55.4C7 52.8 17 52.4 29 52.9C41 53.4 52 55.4 62.5 58.4C52 60.2 41 61.2 29 61.4C17 61.6 7 59.2 7 55.4Z"
        filter={`url(#${shadowBlurId})`}
        className="fill-[rgba(59,36,25,0.3)] dark:fill-[rgba(0,0,0,0.85)]"
      />
      {/* The fire itself, shrunk slightly toward the top of the box to leave
          room for the shadow below and to its right. */}
      <g transform="translate(3.2 0) scale(0.9)">
        {/* A very faint warm glow behind the fire, fading to nothing before the
            edge of the box. */}
        <circle cx="32" cy="38" r="30" fill={`url(#${glowId})`} />
        <g transform="translate(4 -4)">
          <path
            d="M16 18C11 29 5 37 5 45C5 52.5 10.5 58 18 58C25 58 30 53 30 46C30 35 20 31 16 18Z"
            fill={`url(#${gradientId})`}
          />
          <path
            d="M29 8C22 20 14 30 14 41C14 51 21 58 30 58C38 58 42 52 42 45C42 33 33 24 29 8Z"
            fill="#F2A98A"
          />
          <path d={FLAME_PATH} fill="#E07A5F" />
        </g>
        {CORE_FLAMES.map(({ fill, x, sx, sy }) => (
          <path
            key={`${x}-${fill}`}
            d={FLAME_PATH}
            fill={fill}
            transform={`translate(${x} 54) scale(${sx} ${sy}) translate(-35 -58)`}
          />
        ))}
        <path d={WOOD_FILLER} fill="#4F3428" />
        {STICKS.map(({ fill, d }) => (
          <path key={d} d={d} fill={fill} />
        ))}
      </g>
    </svg>
  );
};

export default FireIcon;
