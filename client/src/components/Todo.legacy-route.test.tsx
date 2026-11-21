import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { api, item } from '@/test/todoHarness';
import Todo from './Todo';

/**
 * FS-02JBZ R91: the daily scope is retired. The legacy /plans/[planId] route
 * renders a bare <Todo />; it no longer offers the Daily | Long-term tab buttons
 * and shows the long-term list.
 */

const { fetchChecklist } = api;

// Served by scope, so a daily fetch would surface "Inbox zero".
beforeEach(() => {
  fetchChecklist.mockImplementation(async (_p, scope) =>
    scope === 'longterm'
      ? [item('L1', { description: 'Ship the beta' })]
      : [item('D1', { scope: 'daily', description: 'Inbox zero' })]
  );
});

describe('Todo on the legacy route (no fixedTaskType)', () => {
  it('should show the long-term list', async () => {
    render(<Todo />);

    expect(await screen.findByText('Ship the beta')).toBeTruthy();
    expect(screen.queryByText('Inbox zero')).toBeNull();
    const scopes = fetchChecklist.mock.calls.map((c) => c[1]);
    expect(scopes).not.toContain('daily');
  });

  it('should offer no Daily | Long-term tab buttons', async () => {
    render(<Todo />);
    await screen.findByText('Ship the beta');

    expect(screen.queryByRole('button', { name: 'Daily' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Long-term' })).toBeNull();
  });
});
