"use client";

import { ReactNode, useEffect } from "react";
import { useSidebar } from "./LayoutWrapper";
import Sidebar from "./Sidebar";
import UserProfile from "./UserProfile";
import Logo from "./Logo";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useAuth } from "@/context/AuthContext";
import AuthGuard from "./AuthGuard";
import ThemeToggle from "./ThemeToggle";
import { toast } from "@/components/ui/use-toast";

interface LayoutContentProps {
  children: ReactNode;
}

export default function LayoutContent({ children }: LayoutContentProps) {
  const { isCollapsed, setIsCollapsed } = useSidebar();
  const { isAuthenticated } = useAuth();
  const pathname = usePathname();
  const isHomePage = pathname === "/";
  const isAuthPage = pathname === "/auth";
  const isPlanPage = pathname.startsWith("/plan/");

  // Home page should always start with the sidebar hidden, regardless of
  // whether it was pinned open on a previous page. Users can still slide/pin
  // it in, but entry to home is always clean.
  useEffect(() => {
    if (isHomePage) {
      setIsCollapsed(true);
    }
  }, [isHomePage, setIsCollapsed]);

  // Sidebar hints for plan pages and the authenticated home page.
  // Same localStorage-gated behavior as before: first-timer toast once, then
  // a 24hr reminder — so dismissed hints don't re-appear.
  useEffect(() => {
    if ((!isPlanPage && !isHomePage) || !isAuthenticated) return;

    // First-timer hint
    const hasSeenHint = localStorage.getItem("hasSeenSidebarHint");
    if (!hasSeenHint) {
      setTimeout(() => {
        toast({
          title: "Tip: Your plans live in the side panel",
          description: "← Hover or click the left edge to browse",
          position: "bottom-left",
        });
        localStorage.setItem("hasSeenSidebarHint", "true");
      }, 1500);
      return;
    }

    // 24hr reminder
    const lastOpen = localStorage.getItem("lastSidebarOpen");
    if (!lastOpen) {
      toast({
        title: "Tip: Switch plans from the side panel",
        description: "← Hover or click the left edge",
        position: "bottom-left",
      });
      return;
    }

    const hoursSinceOpen = (Date.now() - new Date(lastOpen).getTime()) / (1000 * 60 * 60);
    if (hoursSinceOpen > 24) {
      toast({
        title: "Tip: Switch plans from the side panel",
        description: "← Hover or click the left edge",
        position: "bottom-left",
      });
    }
  }, [isPlanPage, isHomePage, isAuthenticated]);

  // Auth page always gets minimal layout
  if (isAuthPage) {
    return <>{children}</>;
  }

  // Home page: minimal layout for unauthenticated (splash), full layout for authenticated (dashboard)
  if (isHomePage && !isAuthenticated) {
    return <>{children}</>;
  }

  return (
    <AuthGuard>
      <Sidebar />

      {/* Main content */}
      <div
        className={`transition-all duration-300 ${
          isHomePage ? "ml-0" : isCollapsed ? "ml-0" : "ml-64"
        }`}
      >
        {/* Top bar with logo and user profile */}
        <div
          className="fixed top-0 right-0 h-16 bg-layout backdrop-blur-sm z-10"
          style={{ left: isHomePage ? "0" : isCollapsed ? "0" : "16rem" }}
        >
          <div className="h-full max-w-7xl mx-auto px-8 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Logo />
              <Link href="/">
                <h1 className="text-2xl font-bold">Fireplace</h1>
              </Link>
            </div>
            <div className="flex items-center gap-3">
              <ThemeToggle />
              <UserProfile />
            </div>
          </div>
        </div>

        {/* Main content with top padding to accommodate the top bar */}
        <main className="pt-20 p-8">
          <div className="max-w-7xl mx-auto">{children}</div>
        </main>
      </div>
    </AuthGuard>
  );
}
