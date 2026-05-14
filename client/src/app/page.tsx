"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "@/context/AuthContext";

function LandingPage() {
  return (
    <main className="flex flex-col items-center justify-center min-h-screen bg-layout px-6">
      <div className="max-w-2xl text-center space-y-8">
        <h1
          className="text-5xl md:text-6xl font-bold tracking-tight"
          style={{ color: "rgb(247, 111, 83)" }}
        >
          Fireplace
        </h1>

        <p className="text-xl md:text-2xl font-light leading-relaxed text-[#2e2e2e]/80 dark:text-[#d1cfc0]">
          Sit down by the fire. <br />
          Start your plan now.
        </p>

        <p className="text-sm max-w-md mx-auto text-[#2e2e2e]/50 dark:text-[#d1cfc0]/50">
          A developer and learning platform that houses your checklists, daily
          reviews, learning tracks, and AI-powered suggestions.
        </p>

        <div className="flex items-center justify-center gap-4 pt-4">
          <Link
            href="/auth"
            className="px-8 py-3 rounded-md text-white font-medium transition-colors bg-[rgb(247,111,83)] hover:bg-[rgb(237,101,73)]"
          >
            Sign In
          </Link>
          <Link
            href="/auth?tab=signup"
            className="px-8 py-3 rounded-md font-medium transition-colors text-[rgb(247,111,83)] border border-[rgb(247,111,83)] hover:bg-[rgba(247,111,83,0.1)]"
          >
            Sign Up
          </Link>
        </div>
      </div>
    </main>
  );
}

function Dashboard() {
  const { user } = useAuth();
  const router = useRouter();
  const [selectedType, setSelectedType] = useState<string | null>(null);
  const [focusText, setFocusText] = useState("");
  const [typeError, setTypeError] = useState(false);

  const handleStart = () => {
    if (!selectedType) {
      setTypeError(true);
      return;
    }
    setTypeError(false);

    const params = new URLSearchParams({
      name: focusText.trim(),
      focus: focusText.trim(),
      planType: selectedType,
    });
    router.push(`/create-plan?${params.toString()}`);
  };

  const handleTypeSelect = (type: string) => {
    setSelectedType(type);
    setTypeError(false);
  };

  return (
    <main className="min-h-screen p-8">
      <div className="max-w-7xl mx-auto space-y-12">
        {/* Welcome Section */}
        <div className="backdrop-blur-sm rounded-2xl p-8 shadow-lg bg-white/5 dark:bg-gray-900/10">
          <h1 className="text-4xl font-bold mb-2">
            Welcome back, {user?.name || "there"}.
          </h1>
          <p className="opacity-80">
            Let&apos;s continue your journey.
          </p>
        </div>

        {/* Focus Selection Section */}
        <div className="flex flex-col items-center justify-center min-h-[50vh]">
          <h2 className="text-3xl font-medium text-center mb-12">
            What will be your focus of the day?
          </h2>

          {/* Plan Type Cards */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 w-full max-w-2xl mb-12">
            <button
              onClick={() => handleTypeSelect("project")}
              className={`p-6 rounded-xl backdrop-blur-sm transition-all ${
                selectedType === "project"
                  ? "ring-2 ring-[rgb(247,111,83)] bg-[rgba(247,111,83,0.1)] shadow-lg scale-[1.02]"
                  : "bg-foreground/5 hover:bg-foreground/10"
              }`}
            >
              <h3 className="text-xl font-medium mb-2">Project</h3>
              <p className="text-base opacity-80">
                Focus on building and improving your projects
              </p>
            </button>
            <button
              onClick={() => handleTypeSelect("learning")}
              className={`p-6 rounded-xl backdrop-blur-sm transition-all ${
                selectedType === "learning"
                  ? "ring-2 ring-[rgb(247,111,83)] bg-[rgba(247,111,83,0.1)] shadow-lg scale-[1.02]"
                  : "bg-foreground/5 hover:bg-foreground/10"
              }`}
            >
              <h3 className="text-xl font-medium mb-2">Learning</h3>
              <p className="text-base opacity-80">
                Focus on acquiring new skills and knowledge
              </p>
            </button>
          </div>
          {typeError && (
            <p className="text-red-400 text-base mb-4 -mt-8">
              Please select a plan type first
            </p>
          )}

          {/* Focus Input */}
          <div className="w-full max-w-2xl mb-6">
            <input
              type="text"
              value={focusText}
              onChange={(e) => setFocusText(e.target.value)}
              placeholder="Enter your focus. E.g. building a movie app, learning microservices..."
              className="w-full px-4 py-3 text-xl bg-transparent border-b border-foreground/20 focus:border-[rgb(247,111,83)]/60 outline-none text-foreground placeholder:text-foreground/40 transition-colors"
            />
          </div>

          {/* Start button — always occupies space so nothing jumps when it appears */}
          <button
            onClick={handleStart}
            disabled={!focusText.trim()}
            aria-hidden={!focusText.trim()}
            tabIndex={focusText.trim() ? 0 : -1}
            className={`w-full max-w-2xl py-4 rounded-xl text-lg font-semibold text-white transition-all duration-300 bg-[rgb(247,111,83)] hover:bg-[rgb(237,101,73)] hover:shadow-lg hover:shadow-[rgba(247,111,83,0.3)] active:scale-[0.98] ${
              focusText.trim()
                ? "opacity-100 pointer-events-auto"
                : "opacity-0 pointer-events-none"
            }`}
          >
            Start this plan
          </button>

          {/* Skip Link — kept strategically lower, clear breathing room from the Start button slot */}
          <div className="w-full max-w-2xl mt-12">
            <Link
              href="/myplans"
              className="text-base text-foreground/40 hover:text-foreground/60 transition-colors float-right"
            >
              Skip to plans
            </Link>
          </div>
        </div>
      </div>
    </main>
  );
}

export default function Home() {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-layout">
        <div className="text-base text-[#2e2e2e]/50 dark:text-[#d1cfc0]/50">
          Loading...
        </div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <LandingPage />;
  }

  return <Dashboard />;
}
