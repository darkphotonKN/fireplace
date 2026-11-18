import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { useDocumentTitle } from './useDocumentTitle';

function Titled({ title }: { title: string }) {
  useDocumentTitle(title);
  return null;
}

const settle = () => new Promise((r) => setTimeout(r, 0));

describe('useDocumentTitle', () => {
  it('should name the tab while mounted and give the old name back after', () => {
    document.title = 'Fireplace';
    const view = render(<Titled title="Launch ideas" />);
    expect(document.title).toBe('Launch ideas');
    view.unmount();
    expect(document.title).toBe('Fireplace');
  });

  it("should hold its name against the app's metadata title landing after it", async () => {
    document.title = 'Fireplace';
    const view = render(<Titled title="Launch ideas" />);
    // Next streams the root metadata in after the page's effects have run.
    document.title = 'Fireplace';
    await settle();
    expect(document.title).toBe('Launch ideas');
    view.unmount();
  });

  it('should give back the metadata title that lands after a direct load, not the empty one before it', async () => {
    // A direct load of the page: its effect runs before Next has streamed
    // the root metadata <title> in, so the tab has no name yet.
    document.title = '';
    const view = render(<Titled title="Launch ideas" />);
    document.title = 'Fireplace';
    await settle();
    expect(document.title).toBe('Launch ideas');
    view.unmount();
    expect(document.title).toBe('Fireplace');
  });

  it('should never blank the tab on the way out', () => {
    document.title = '';
    const view = render(<Titled title="Launch ideas" />);
    view.unmount();
    expect(document.title).not.toBe('');
  });
});
