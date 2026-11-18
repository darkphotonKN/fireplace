import { describe, it, expect } from 'vitest';
import { linkify } from './linkify';

describe('linkify (FS-PGG2N R23, G4)', () => {
  it('should leave text with no URL as one plain piece', () => {
    expect(linkify('just words\nand a line')).toEqual([
      { text: 'just words\nand a line' },
    ]);
  });

  it('should make http and https URLs links, keeping the text around them', () => {
    expect(linkify('see https://go.dev/ref/mem and http://a.b/c?d=1#e too')).toEqual([
      { text: 'see ' },
      { text: 'https://go.dev/ref/mem', href: 'https://go.dev/ref/mem' },
      { text: ' and ' },
      { text: 'http://a.b/c?d=1#e', href: 'http://a.b/c?d=1#e' },
      { text: ' too' },
    ]);
  });

  it('should leave trailing sentence punctuation out of the link', () => {
    expect(linkify('Read https://go.dev/doc. Then (https://go.dev/blog), ok')).toEqual([
      { text: 'Read ' },
      { text: 'https://go.dev/doc', href: 'https://go.dev/doc' },
      { text: '. Then (' },
      { text: 'https://go.dev/blog', href: 'https://go.dev/blog' },
      { text: '), ok' },
    ]);
  });

  it('should keep a closing bracket the URL itself opened (FS-65B3N R27)', () => {
    expect(linkify('see https://en.wikipedia.org/wiki/Go_(language)!')).toEqual([
      { text: 'see ' },
      {
        text: 'https://en.wikipedia.org/wiki/Go_(language)',
        href: 'https://en.wikipedia.org/wiki/Go_(language)',
      },
      { text: '!' },
    ]);
    expect(linkify('(see https://a.b/x_(y)).')).toEqual([
      { text: '(see ' },
      { text: 'https://a.b/x_(y)', href: 'https://a.b/x_(y)' },
      { text: ').' },
    ]);
  });

  it('should keep markup as the literal characters it is', () => {
    expect(linkify('<b>x</b> https://a.b/<i>')).toEqual([
      { text: '<b>x</b> ' },
      { text: 'https://a.b/<i>', href: 'https://a.b/<i>' },
    ]);
  });

  it.each([
    'javascript:alert(1)',
    'mailto:me@example.com',
    'ftp://files.example.com',
    'data:text/html,<b>x</b>',
  ])('should keep %s as plain text', (text) => {
    expect(linkify(text)).toEqual([{ text }]);
  });

  it('should not link a scheme-only fragment', () => {
    expect(linkify('https:// nothing')).toEqual([{ text: 'https:// nothing' }]);
  });

  it('should give back empty for empty text', () => {
    expect(linkify('')).toEqual([]);
  });
});
