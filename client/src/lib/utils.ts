import { type ClassValue, clsx } from 'clsx';
import { extendTailwindMerge } from 'tailwind-merge';

// Teach the merge about the theme's own shape utilities (tailwind.config.js),
// so `cn('rounded-tile', 'rounded-2xl')` keeps the last one as it would for
// any built-in size, instead of keeping both and leaving it to CSS order.
const twMerge = extendTailwindMerge({
  extend: {
    classGroups: {
      rounded: [{ rounded: ['control', 'tile'] }],
      shadow: [{ shadow: ['tile'] }],
    },
  },
});

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
