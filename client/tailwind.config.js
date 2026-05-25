/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: ['class'],
  content: [
    './pages/**/*.{ts,tsx}',
    './components/**/*.{ts,tsx}',
    './app/**/*.{ts,tsx}',
    './src/**/*.{ts,tsx}',
  ],
  theme: {
    container: {
      center: true,
      padding: '2rem',
      screens: {
        '2xl': '1400px',
      },
    },
    extend: {
      // Every Tailwind text-* utility is bumped +2px from its default size
      // (lineHeight bumped proportionally). This applies globally: card titles,
      // inputs, placeholders, body text, badges, etc. — every text element
      // inside a card or anywhere else picks up the new scale automatically.
      // Defaults were: xs 12/16, sm 14/20, base 16/24, lg 18/28, xl 20/28,
      //                2xl 24/32, 3xl 30/36, 4xl 36/40, 5xl 48, 6xl 60.
      fontSize: {
        xs: ['14px', { lineHeight: '18px' }],
        sm: ['16px', { lineHeight: '22px' }],
        base: ['18px', { lineHeight: '26px' }],
        lg: ['20px', { lineHeight: '30px' }],
        xl: ['22px', { lineHeight: '30px' }],
        '2xl': ['26px', { lineHeight: '34px' }],
        '3xl': ['32px', { lineHeight: '38px' }],
        '4xl': ['38px', { lineHeight: '42px' }],
        '5xl': ['50px', { lineHeight: '1' }],
        '6xl': ['62px', { lineHeight: '1' }],
        '7xl': ['74px', { lineHeight: '1' }],
        '8xl': ['98px', { lineHeight: '1' }],
        '9xl': ['130px', { lineHeight: '1' }],
      },
      colors: {
        border: 'hsl(var(--border))',
        input: 'hsl(var(--input))',
        ring: 'hsl(var(--ring))',
        background: 'hsl(var(--background))',
        foreground: 'hsl(var(--foreground))',
        primary: {
          DEFAULT: 'hsl(var(--primary))',
          foreground: 'hsl(var(--primary-foreground))',
        },
        secondary: {
          DEFAULT: 'hsl(var(--secondary))',
          foreground: 'hsl(var(--secondary-foreground))',
        },
        destructive: {
          DEFAULT: 'hsl(var(--destructive))',
          foreground: 'hsl(var(--destructive-foreground))',
        },
        muted: {
          DEFAULT: 'hsl(var(--muted))',
          foreground: 'hsl(var(--muted-foreground))',
        },
        accent: {
          DEFAULT: 'hsl(var(--accent))',
          foreground: 'hsl(var(--accent-foreground))',
        },
        popover: {
          DEFAULT: 'hsl(var(--popover))',
          foreground: 'hsl(var(--popover-foreground))',
        },
        card: {
          DEFAULT: 'hsl(var(--card))',
          foreground: 'hsl(var(--card-foreground))',
        },
      },
      borderRadius: {
        lg: 'var(--radius)',
        md: 'calc(var(--radius) - 2px)',
        sm: 'calc(var(--radius) - 4px)',
      },
      keyframes: {
        'accordion-down': {
          from: { height: 0 },
          to: { height: 'var(--radix-accordion-content-height)' },
        },
        'accordion-up': {
          from: { height: 'var(--radix-accordion-content-height)' },
          to: { height: 0 },
        },
        // Entrance animation for newly added checklist items.
        // Slides down + fades in + a soft orange "halo" pulse so the
        // user sees exactly where the new task landed in the list.
        fadeIn: {
          '0%': {
            opacity: '0',
            transform: 'translateY(-8px)',
            boxShadow: '0 0 0 0 rgba(247, 111, 83, 0)',
          },
          '40%': {
            opacity: '1',
            transform: 'translateY(0)',
            boxShadow: '0 0 0 4px rgba(247, 111, 83, 0.18)',
          },
          '100%': {
            opacity: '1',
            transform: 'translateY(0)',
            boxShadow: '0 0 0 0 rgba(247, 111, 83, 0)',
          },
        },
      },
      animation: {
        'accordion-down': 'accordion-down 0.2s ease-out',
        'accordion-up': 'accordion-up 0.2s ease-out',
        // ease-out-quint curve for a modern, snappy settle
        fadeIn: 'fadeIn 600ms cubic-bezier(0.22, 1, 0.36, 1) both',
      },
    },
  },
  plugins: [require('tailwindcss-animate')],
};
