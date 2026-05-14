'use client';

import { createContext, useContext, useState, ReactNode } from 'react';
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

export default function LayoutWrapper({ children }: LayoutWrapperProps) {
  const [isCollapsed, setIsCollapsed] = useState(true);

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