"use client";

import { ReactNode, useEffect } from "react";
import Sidebar from "./Sidebar";
import UserProfile from "./UserProfile";
import Logo from "./Logo";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useAuth } from "@/context/AuthContext";
import AuthGuard from "./AuthGuard";
import ThemeToggle from "./ThemeToggle";
import { toast } from "@/components/ui/use-toast";
import { getSidebarState } from "@/lib/sidebarState";

/** The tip's own flags; blocked storage reads as unset and drops writes (FS-S6WVY R8). */
function readFlag(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function writeFlag(key: string, value: string) {
  try {
    localStorage.setItem(key, value);
  } catch {
    // Blocked: the tip may show again next time, which is harmless.
  }
}

interface LayoutContentProps {
  children: ReactNode;
}

export default function LayoutContent({ children }: LayoutContentProps) {
  const { isAuthenticated } = useAuth();
  const pathname = usePathname();
  const isHomePage = pathname === "/";
  const isAuthPage = pathname === "/auth";
  const isPlanPage = pathname.startsWith("/plan/");

  // Sidebar hints for plan pages only (FS-KSJFR R46).
  //
  // This used to fire on the authenticated home too. Home now lists plans on
  // the page — Continue cards and Your other plans — so a tip pointing at a
  // panel that shows what the user can already see reads as an unfinished
  // redesign.
  //
  // Nor does it fire when the sidebar is already open (FS-S6WVY R12): the tip
  // would point at a panel the user can see, so it is skipped and the
  // first-timer flag is left unspent. It is decided on arrival from the
  // remembered state, never from the context, whose pre-hydration value is
  // always open; toggling on the page doesn't re-run it (R13).
  //
  // Same localStorage-gated behavior as before: first-timer toast once, then
  // a 24hr reminder — so dismissed hints don't re-appear.
  useEffect(() => {
    if (!isPlanPage || !isAuthenticated) return;
    if (getSidebarState() !== "hidden") return;

    // First-timer hint
    const hasSeenHint = readFlag("hasSeenSidebarHint");
    if (!hasSeenHint) {
      setTimeout(() => {
        toast({
          title: "Tip: Your plans live in the side panel",
          description: "← Hover or click the left edge to browse",
          position: "bottom-left",
        });
        writeFlag("hasSeenSidebarHint", "true");
      }, 1500);
      return;
    }

    // 24hr reminder
    const lastOpen = readFlag("lastSidebarOpen");
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
  }, [isPlanPage, isAuthenticated]);

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

      {/* The top bar spans the viewport whatever the sidebar does, so the logo
          and title never move (FS-S6WVY R14). Its bottom hairline and the
          sidebar's right one frame the page as one surface (R15). */}
      <header className="fixed inset-x-0 top-0 z-10 h-16 border-b border-border bg-layout">
        {/* pl-2.5 centres the logo on the rail, over the sidebar toggle. */}
        <div className="flex h-full items-center justify-between pl-2.5 pr-4">
          <div className="flex items-center gap-2 text-2xl">
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
      </header>

      {/* Main content. Its offset follows the docked sidebar's width (open or
          the rail) through `<html data-sidebar>` in globals.css, so it is right
          on the first frame, the same on every route, `/` included, and moves
          in step with the sidebar (FS-S6WVY R5, R18, R19). */}
      <div className="fp-sidebar-offset">
        {/* Main content with top padding to accommodate the top bar */}
        <main className="pt-20 p-8">
          <div className="max-w-7xl mx-auto">{children}</div>
        </main>
      </div>
    </AuthGuard>
  );
}
