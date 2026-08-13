import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { mockMatchMedia, mockIntersectionObserver } from '@/test/motion';

const auth = { user: null as { name: string } | null, isAuthenticated: false, isLoading: true };

vi.mock('@/context/AuthContext', () => ({ useAuth: () => auth }));
vi.mock('next/navigation', () => ({ useRouter: () => ({ push: vi.fn() }) }));
// The chrome bar reuses the app's ThemeToggle, so the tour needs the theme
// context. Everything else in the landing renders provider-free.
vi.mock('@/context/ThemeContext', () => ({
  useTheme: () => ({ theme: 'dark', toggleTheme: vi.fn() }),
}));

import Home from './page';

describe('Home', () => {
  beforeEach(() => {
    mockMatchMedia({ wide: true });
    mockIntersectionObserver();
    auth.user = null;
    auth.isAuthenticated = false;
    auth.isLoading = true;
  });

  it('should paint the hero while auth is still resolving', () => {
    render(<Home />);

    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Fireplace');
    expect(screen.queryByText(/loading/i)).toBeNull();
  });

  it('should show the dashboard once the visitor is known to be authenticated', () => {
    auth.isAuthenticated = false;
    auth.isLoading = false;
    const { rerender } = render(<Home />);
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Fireplace');

    auth.isAuthenticated = true;
    auth.user = { name: 'Kranti' };
    rerender(<Home />);

    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent(/welcome back/i);
  });
});
