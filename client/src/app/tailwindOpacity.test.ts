import { describe, it, expect } from 'vitest';
import { createRequire } from 'node:module';
import { readdirSync, readFileSync, statSync } from 'node:fs';
import path from 'node:path';
import postcss from 'postcss';
import tailwindcss from 'tailwindcss';

// A colour utility with an opacity modifier (`text-foreground/88`) compiles to
// nothing when the number is off Tailwind's opacity scale, and nothing warns:
// the text just renders at full strength. This compiles every such class the
// source uses against the real config and fails on any that yields no rule.

const require = createRequire(import.meta.url);
const config = require('../../tailwind.config.js');

const SRC = path.resolve(__dirname, '..');
const OPACITY_CLASS =
  /\b(?:text|bg|border|ring|fill|stroke|from|via|to|shadow|outline|divide|placeholder|decoration|caret|accent)-[a-z]+(?:-[a-z]+)*\/\d+\b/g;

function sourceFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const full = path.join(dir, name);
    if (statSync(full).isDirectory()) return name === 'generated' ? [] : sourceFiles(full);
    return /\.tsx?$/.test(name) && !/\.test\.tsx?$/.test(name) ? [full] : [];
  });
}

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

  it('generates every opacity-modified colour class the source uses', async () => {
    const used = new Set<string>();
    for (const file of sourceFiles(SRC)) {
      for (const m of readFileSync(file, 'utf8').matchAll(OPACITY_CLASS)) used.add(m[0]);
    }
    expect(used.size).toBeGreaterThan(0);
    const css = await compile([...used]);
    const missing = [...used].filter((cls) => !css.includes(selector(cls)));
    expect(missing).toEqual([]);
  });
});
