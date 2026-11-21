import { describe, it, expect } from 'vitest';
import { createRequire } from 'node:module';
import postcss from 'postcss';
import tailwindcss from 'tailwindcss';

// A colour utility with an opacity modifier (`text-foreground/88`) compiles to
// nothing when the number is off Tailwind's opacity scale, and nothing warns:
// the text just renders at full strength. This compiles the note text colour
// against the real config, so the scale step it needs can't be dropped.

const require = createRequire(import.meta.url);
const config = require('../../tailwind.config.js');

async function compile(classes: string[]): Promise<string> {
  const result = await postcss([
    tailwindcss({ ...config, content: [{ raw: classes.join(' '), extension: 'html' }] }),
  ]).process('@tailwind utilities;', { from: undefined });
  return result.css;
}

const selector = (cls: string) => '.' + cls.replace('/', '\\/') + ' ';

describe('tailwind opacity modifiers', () => {
  it('generates text-foreground/88, the note text colour (FS-PGG2N R47)', async () => {
    const css = await compile(['text-foreground/88']);
    expect(css).toContain(selector('text-foreground/88'));
    expect(css).toMatch(/color:\s*hsl\(var\(--foreground\)\s*\/\s*0\.88\)/);
  });
});
