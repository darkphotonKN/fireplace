'use client';

import { useState, useRef, useEffect } from 'react';
import Link from 'next/link';
import { getProfile } from '@/services/api';
import { useAuth } from '@/context/AuthContext';

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
    <div className="relative z-50" ref={dropdownRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center space-x-2 text-foreground/80 hover:text-foreground transition-colors"
      >
        <span className="text-sm">{displayName}</span>
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
          className={`w-3 h-3 transition-transform ${
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
