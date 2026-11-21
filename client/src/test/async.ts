import { act } from '@testing-library/react';

/** A promise a test resolves (or rejects) by hand, so "in flight" is an observable state. */
export function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

/**
 * Let everything already scheduled run, renders included.
 *
 * An assertion about what does NOT happen (a request budget, no second fetch)
 * can't be made with `waitFor`: it resolves the instant the awaited state is
 * reached, so a call issued on a later tick lands after the test has looked.
 * Settling first gives it its chance to happen and be counted.
 */
export const settle = (ms = 32) =>
  act(async () => {
    await new Promise((resolve) => setTimeout(resolve, ms));
  });
