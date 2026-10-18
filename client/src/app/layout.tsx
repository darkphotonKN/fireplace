import type { Metadata } from 'next';
import { Lora } from 'next/font/google';
import './globals.css';
import LayoutWrapper from '@/components/LayoutWrapper';
import LayoutContent from '@/components/LayoutContent';
import { SIDEBAR_HEAD_SCRIPT } from '@/lib/sidebarState';

// Lora is a variable font, so there's no weight list: body text at 400 and
// headings at 700 all come from the one file.
const lora = Lora({
  subsets: ['latin'],
  // Note rows and the add-bar placeholder are italic; without this next/font
  // ships only the roman and the browser fakes a slant.
  style: ['normal', 'italic'],
  display: 'swap',
  variable: '--font-lora',
});

export const metadata: Metadata = {
  title: 'Fireplace',
  description:
    'Organize your learning journey and development projects in one place',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className={lora.variable} suppressHydrationWarning>
      <head>
        <script
          dangerouslySetInnerHTML={{
            __html: `
              (function() {
                var theme = localStorage.getItem('theme') || 'dark';
                if (theme === 'dark') {
                  document.documentElement.classList.add('dark');
                }
              })();
            `,
          }}
        />
        {/* Paint the remembered sidebar state before first paint (FS-S6WVY R5). */}
        <script dangerouslySetInnerHTML={{ __html: SIDEBAR_HEAD_SCRIPT }} />
      </head>
      <body className={lora.className}>
        <LayoutWrapper>
          <LayoutContent>{children}</LayoutContent>
        </LayoutWrapper>
      </body>
    </html>
  );
}
