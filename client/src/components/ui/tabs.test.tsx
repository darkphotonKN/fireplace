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
    signIn: screen.getByRole('tab', { name: 'Sign In' }),
    signUp: screen.getByRole('tab', { name: 'Sign Up' }),
  };
}

describe('Tabs underline variant', () => {
  it('should underline the active tab only', () => {
    const { signIn, signUp } = renderTabs('underline');

    expect(signIn.className).toContain('after:scale-x-100');
    expect(signUp.className).toContain('after:scale-x-0');
  });

  it('should move the underline when another tab is chosen', () => {
    const { signIn, signUp } = renderTabs('underline');

    fireEvent.click(signUp);

    expect(screen.getByText('signup panel')).toBeTruthy();
    expect(signUp.className).toContain('after:scale-x-100');
    expect(signIn.className).toContain('after:scale-x-0');
  });

  it('should keep the filled segment on the active tab in the default strip', () => {
    const { signIn, signUp } = renderTabs();

    expect(signIn.className).toContain('bg-gray-700');
    expect(signUp.className).not.toContain('bg-gray-700');
  });
});
