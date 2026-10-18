'use client';

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useSyncExternalStore,
  ReactNode,
} from 'react';
import { AuthProvider } from '@/context/AuthContext';
import { ThemeProvider } from '@/context/ThemeContext';
import Toaster from '@/components/ui/Toaster';
import {
  chooseSidebarState,
  getServerSidebarState,
  getSidebarState,
  markSidebarReady,
  subscribeSidebarState,
} from '@/lib/sidebarState';

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

export default function LayoutWrapper({ children }: LayoutWrapperProps) {
  // Whether the sidebar is pinned open or hidden, remembered on this device
  // (FS-S6WVY R1-R8). The server snapshot is open; the head script has
  // already painted the real state, so hydration matches and nothing flashes.
  const state = useSyncExternalStore(
    subscribeSidebarState,
    getSidebarState,
    getServerSidebarState,
  );

  // Let the sidebar animate only once the first frame is up, so a load that
  // is hidden by preference doesn't slide closed (R6).
  useEffect(() => {
    const frame = requestAnimationFrame(markSidebarReady);
    return () => cancelAnimationFrame(frame);
  }, []);

  // Only the pin/hide toggle calls this: it is the one writer of the choice.
  // Hovering a hidden sidebar open is local to Sidebar and never remembered.
  const setIsCollapsed = useCallback((collapsed: boolean) => {
    chooseSidebarState(collapsed ? 'hidden' : 'open');
  }, []);

  const sidebar = useMemo(
    () => ({ isCollapsed: state === 'hidden', setIsCollapsed }),
    [state, setIsCollapsed],
  );

  return (
    <ThemeProvider>
      <AuthProvider>
        <SidebarContext.Provider value={sidebar}>
          <div className="min-h-screen bg-layout">
            {children}
            <Toaster />
          </div>
        </SidebarContext.Provider>
      </AuthProvider>
    </ThemeProvider>
  );
}