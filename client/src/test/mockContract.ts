import { expect } from 'vitest';

/**
 * Asserts the landing mock contract on every mock rendered in `container`.
 *
 * Every tour section that carries mocked art calls this, so the rules live in
 * one place and a new section inherits the coverage by using the helper rather
 * than by remembering the rules.
 */
export function expectMockContract(container: HTMLElement) {
  const mocks = container.querySelectorAll('[data-landing-mock]');
  expect(mocks.length).toBeGreaterThan(0);

  mocks.forEach((mock) => {
    // Decorative: the copy is the narration, not a recital of invented labels.
    expect(mock).toHaveAttribute('aria-hidden', 'true');
    // Fake controls are never real ones, and nothing here is reachable by tab.
    expect(mock.querySelectorAll('input, button, a, [tabindex], [role]')).toHaveLength(0);
  });
}
