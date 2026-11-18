import { useEffect } from 'react';

/**
 * Names the tab after what a page shows while it is open, so a bookmark is
 * too, and gives the old title back on the way out (FS-043WP R12, FS-PGG2N
 * R21).
 *
 * It holds the name, rather than setting it once: on a full load Next streams
 * the root layout's metadata `<title>` in after the page's effects have run,
 * which would otherwise put "Fireplace" back over it. On such a load the tab
 * has no name yet when the hook first looks, so the name to give back is the
 * first one that is not ours to land, and an empty one is never given back.
 */
export function useDocumentTitle(title: string) {
  useEffect(() => {
    let previous = document.title;
    const hold = () => {
      if (document.title === title) return;
      if (!previous) previous = document.title;
      document.title = title;
    };
    hold();
    const watch = new MutationObserver(hold);
    watch.observe(document.head, {
      subtree: true,
      childList: true,
      characterData: true,
    });
    return () => {
      watch.disconnect();
      if (previous) document.title = previous;
    };
  }, [title]);
}
