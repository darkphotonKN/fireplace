import { linkify } from '@/lib/linkify';

/**
 * Plain text with its http(s) URLs as links that open a new tab (FS-PGG2N
 * R23, FS-65B3N R27). Rendered as React text and anchors, never as HTML, so
 * markup in the text shows as the characters it is.
 */
export default function LinkedText({ text, rel }: { text: string; rel: string }) {
  return (
    <>
      {linkify(text).map((piece, i) =>
        piece.href ? (
          <a
            key={i}
            href={piece.href}
            target="_blank"
            rel={rel}
            className="text-primary underline decoration-primary/40 underline-offset-2 hover:decoration-primary"
          >
            {piece.text}
          </a>
        ) : (
          <span key={i}>{piece.text}</span>
        )
      )}
    </>
  );
}
