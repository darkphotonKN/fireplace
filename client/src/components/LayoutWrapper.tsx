'use client';

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  ReactNode,
} from 'react';
import { AuthProvider } from '@/context/AuthContext';
import { ThemeProvider } from '@/context/ThemeContext';
import Toaster from '@/components/ui/Toaster';

interface SidebarContextType {
  isCollapsed: boolean;
  setIsCollapsed: (collapsed: boolean) => void;
}

const SidebarContext = createContext<SidebarContextType>({
  isCollapsed: false,
  setIsCollapsed: () => {},
});

export const useSidebar = () => useContext(SidebarContext);

interface LayoutWrapperProps {
  children: ReactNode;
}

// Whether the sidebar is pinned open or hidden, remembered on this device.
// Only the pin/hide toggle writes it — hovering a collapsed sidebar open is
// local to the sidebar and deliberately not remembered.
const SIDEBAR_KEY = 'sidebarCollapsed';

export default function LayoutWrapper({ children }: LayoutWrapperProps) {
  const [isCollapsed, setCollapsed] = useState(true);

  // Read on mount rather than in the initializer: there is no storage during
  // the server render. Nothing stored keeps the collapsed default.
  useEffect(() => {
    try {
      const saved = window.localStorage.getItem(SIDEBAR_KEY);
      if (saved !== null) setCollapsed(saved === 'true');
    } catch {
      // Storage blocked; the sidebar still works, it just won't be remembered.
    }
  }, []);

  const setIsCollapsed = useCallback((collapsed: boolean) => {
    setCollapsed(collapsed);
    try {
      window.localStorage.setItem(SIDEBAR_KEY, String(collapsed));
    } catch {
      // As above: not remembered this time.
    }
  }, []);

  return (
    <ThemeProvider>
      <AuthProvider>
        <SidebarContext.Provider value={{ isCollapsed, setIsCollapsed }}>
          <div className="min-h-screen bg-layout">
            {children}
            <Toaster />
          </div>
        </SidebarContext.Provider>
      </AuthProvider>
    </ThemeProvider>
  );
}