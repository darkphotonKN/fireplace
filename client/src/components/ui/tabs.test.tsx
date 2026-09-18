import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { Tabs, TabsList, TabsTrigger, TabsContent } from './tabs';

/**
 * The `underline` variant is opt-in: the auth page uses it, while the notes and
 * checklist strips keep the original segmented control.
 */
function renderTabs(variant?: 'solid' | 'underline') {
  render(
    <Tabs defaultValue="signin">
      <TabsList variant={variant}>
        <TabsTrigger value="signin">Sign In</TabsTrigger>
        <TabsTrigger value="signup">Sign Up</TabsTrigger>
      </TabsList>
      <TabsContent value="signin">signin panel</TabsContent>
      <TabsContent value="signup">signup panel</TabsContent>
    </Tabs>
  );
  return {
    list: screen.getByRole('tab', { name: 'Sign In' }).parentElement!,
    signIn: screen.getByRole('tab', { name: 'Sign In' }),
    signUp: screen.getByRole('tab', { name: 'Sign Up' }),
  };
}

describe('Tabs underline variant', () => {
  it('should drop the filled box for a hairline, and mark the active tab in coral', () => {
    const { list, signIn, signUp } = renderTabs('underline');

    expect(list.className).not.toContain('bg-gray-800');
    expect(list.className).toContain('border-b');

    expect(signIn.className).toContain('text-primary');
    expect(signIn.className).toContain('after:scale-x-100');
    expect(signIn.className).not.toContain('bg-gray-700');

    expect(signUp.className).toContain('text-foreground/40');
    expect(signUp.className).toContain('after:scale-x-0');
  });

  it('should move the underline when another tab is chosen', () => {
    const { signIn, signUp } = renderTabs('underline');

    fireEvent.click(signUp);

    expect(screen.getByText('signup panel')).toBeTruthy();
    expect(signUp.className).toContain('after:scale-x-100');
    expect(signIn.className).toContain('after:scale-x-0');
  });

  it('should leave the default segmented strip unchanged', () => {
    const { list, signIn, signUp } = renderTabs();

    expect(list.className).toContain('bg-gray-800');
    expect(signIn.className).toContain('bg-gray-700');
    expect(signUp.className).toContain('text-gray-400');
  });
});
