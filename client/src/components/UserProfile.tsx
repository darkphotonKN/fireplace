'use client';

import { useState, useRef, useEffect } from 'react';
import Link from 'next/link';
import { getProfile } from '@/services/api';
import { useAuth } from '@/context/AuthContext';
import { FOCUS_RING } from '@/lib/chrome';
import { cn } from '@/lib/utils';

export default function UserProfile() {
  const [isOpen, setIsOpen] = useState(false);
  const [userName, setUserName] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const { signOut } = useAuth();

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node)
      ) {
        setIsOpen(false);
      }
    }

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  useEffect(() => {
    getProfile()
      .then((res) => {
        setUserName(res.displayName || res.name);
      })
      .catch(() => {
        setUserName('User');
      })
      .finally(() => setLoading(false));
  }, []);

  const handleSignOut = () => {
    setIsOpen(false);
    signOut();
  };

  const displayName = loading ? '...' : (userName || 'User');

  return (
    <div className="relative z-50 min-w-0" ref={dropdownRef}>
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className={cn(
          // A 40px tap target below md, the compact desktop height from md (FS-49CQ2 R17).
          'flex h-10 min-w-10 max-w-full items-center justify-center gap-2 rounded-md px-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground md:h-8',
          FOCUS_RING,
        )}
      >
        {/* Below sm the name is read, not shown, so a long one can't push the
            bar past a 375px phone; it stays the button's name (R18). From sm
            it shows as before, cut short only if the bar runs out of room. */}
        <span className="sr-only text-sm sm:not-sr-only sm:truncate">{displayName}</span>
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
          className={`w-3 h-3 shrink-0 transition-transform ${
            isOpen ? 'rotate-180' : ''
          }`}
        >
          <path
            fillRule="evenodd"
            d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z"
            clipRule="evenodd"
          />
        </svg>
      </button>

      {isOpen && (
        <div className="absolute right-0 mt-2 w-48 rounded-md shadow-lg bg-[#ebe8d8] dark:bg-[#2a2a2a] backdrop-blur-sm border border-foreground/10">
          <div className="py-1" role="menu" aria-orientation="vertical">
            <Link
              href="/profile"
              className="block px-4 py-2 text-sm text-foreground/80 hover:text-foreground hover:bg-foreground/10"
              role="menuitem"
              onClick={() => setIsOpen(false)}
            >
              Profile
            </Link>
            <Link
              href="/myplans"
              className="block px-4 py-2 text-sm text-foreground/80 hover:text-foreground hover:bg-foreground/10"
              role="menuitem"
              onClick={() => setIsOpen(false)}
            >
              My Plans
            </Link>
            <button
              onClick={handleSignOut}
              className="block w-full text-left px-4 py-2 text-sm text-foreground/80 hover:text-foreground hover:bg-foreground/10"
              role="menuitem"
            >
              Sign Out
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
