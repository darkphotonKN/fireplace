'use client';

import { useLayoutEffect, useRef } from 'react';
import { useRouter } from 'next/navigation';

/**
 * Holds in-app navigations away from a page while the page decides whether
 * to let them go (FS-PGG2N R27, R32): the note page flushes its body first,
 * and may ask for a summary.
 *
 * Next's App Router has no way to block a navigation, so the guard catches
 * the click itself. A capture listener on the window runs before the link's
 * own handler (Next's `<Link>` included) and before the browser follows it.
 * Only a plain left click on a same-origin link to another page is held:
 * modified and middle clicks, new-tab and download links, other origins and
 * hash jumps on this page are left alone. Browser back and forward, reload,
 * and closing the tab are never touched, and no `beforeunload` is added.
 *
 * `decide` gets the destination (path, query and hash) and resolves true to
 * go there, false to stay. While it runs, further clicks are swallowed.
 */
export function useLeaveGuard(decide: (href: string) => Promise<boolean>): void {
  const router = useRouter();
  const latest = useRef({ decide, router });
  latest.current = { decide, router };

  // Listening from the commit itself: a passive effect runs a task later,
  // and a click in that gap would leave unflushed and unasked.
  useLayoutEffect(() => {
    let deciding = false;
    const onClick = (e: MouseEvent) => {
      const href = heldDestination(e);
      if (href === null) return;
      e.preventDefault();
      e.stopPropagation();
      if (deciding) return;
      deciding = true;
      latest.current.decide(href).then(
        (go) => {
          deciding = false;
          if (go) latest.current.router.push(href);
        },
        (err) => {
          deciding = false;
          console.error('Leave guard failed:', err);
        }
      );
    };
    window.addEventListener('click', onClick, true);
    return () => window.removeEventListener('click', onClick, true);
  }, []);
}

/** Where a click is going, if it is an in-app navigation to hold; else null. */
function heldDestination(e: MouseEvent): string | null {
  if (e.defaultPrevented || e.button !== 0) return null;
  if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return null;
  const anchor = (e.target as Element | null)?.closest?.('a[href]');
  if (!(anchor instanceof HTMLAnchorElement)) return null;
  const target = anchor.getAttribute('target');
  if ((target && target !== '_self') || anchor.hasAttribute('download')) return null;
  const url = new URL(anchor.href, window.location.href);
  if (url.origin !== window.location.origin) return null;
  const here = window.location;
  if (url.pathname === here.pathname && url.search === here.search) return null;
  return url.pathname + url.search + url.hash;
}
