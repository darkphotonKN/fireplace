import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';

let pathname = '/';
vi.mock('next/navigation', () => ({ usePathname: () => pathname }));

import HomeButton from './HomeButton';

describe('HomeButton', () => {
  it('links to the dashboard from a plan page, not marked current', () => {
    pathname = '/plan/abc';
    render(<HomeButton />);
    const link = screen.getByRole('link', { name: 'Home' });
    expect(link.getAttribute('href')).toBe('/');
    expect(link.getAttribute('aria-current')).toBeNull();
  });

  it('is marked current on home itself', () => {
    pathname = '/';
    render(<HomeButton />);
    expect(screen.getByRole('link', { name: 'Home' }).getAttribute('aria-current')).toBe('page');
  });
});
