/**
 * A note body split into plain text and links (FS-PGG2N R23, G4). Only
 * `http://` and `https://` URLs become links; every other scheme, from
 * `javascript:` to `mailto:`, stays text. The pieces are rendered as React
 * text and anchors, so nothing is ever injected as HTML.
 */
export interface BodyPiece {
  text: string;
  /** Set only on a link: the URL it opens. */
  href?: string;
}

// A URL runs to the next whitespace. Sentence punctuation that ends it is left
// outside, so "see https://go.dev." links https://go.dev. A closing bracket
// stays only when the URL opened it: "(https://go.dev)" leaves the ")" out,
// while "https://en.wikipedia.org/wiki/Go_(language)" keeps it (FS-65B3N R27).
const URL = /https?:\/\/[^\s]+/g;
const PUNCTUATION = new Set(['.', ',', ';', ':', '!', '?', "'", '"']);
const CLOSERS: Record<string, string> = { ')': '(', ']': '[', '}': '{' };

const count = (s: string, c: string) => s.split(c).length - 1;

function trimTrailing(url: string): string {
  let end = url.length;
  while (end > 0) {
    const last = url[end - 1];
    const opener = CLOSERS[last];
    const head = url.slice(0, end);
    if (PUNCTUATION.has(last) || (opener && count(head, opener) < count(head, last))) {
      end--;
    } else {
      break;
    }
  }
  return url.slice(0, end);
}

export function linkify(text: string): BodyPiece[] {
  const pieces: BodyPiece[] = [];
  let plain = '';
  let last = 0;
  for (const match of text.matchAll(URL)) {
    const url = trimTrailing(match[0]);
    const start = match.index ?? 0;
    // "https://" with nothing after it is not a place to go.
    if (url.length <= url.indexOf('//') + 2) continue;
    plain += text.slice(last, start);
    if (plain) pieces.push({ text: plain });
    plain = '';
    pieces.push({ text: url, href: url });
    last = start + url.length;
  }
  plain += text.slice(last);
  if (plain) pieces.push({ text: plain });
  return pieces;
}
