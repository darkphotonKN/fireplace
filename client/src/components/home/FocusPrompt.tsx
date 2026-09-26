"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/context/AuthContext";

/**
 * The focus prompt — the new-plan funnel that opens `/` (FS-KSJFR R7–R10).
 *
 * The funnel itself is untouched by this feature: plan type, a focus line, and
 * a start action that hands `name`, `focus` and `planType` to `/create-plan`.
 *
 * Two things did change. The greeting moved here from a welcome card, because
 * this is the day's opening ritual and the only surface that greets by name
 * (R8/D8) — and the card chrome it sat in did not come with it (R18). And the
 * skip affordance no longer goes to `/myplans`: it reveals the dashboard in
 * place, so skipping is progress rather than a detour to another index page.
 *
 * Whether this renders at all is the day gate's decision, which is I-KSJFR-6.
 * Today it opens on every visit.
 *
 * Retokenized off the raw `rgb(247,111,83)` literals this route carried
 * (R47–R50). The coral *this component chooses* is the start action (R49); two
 * more are coral by the guideline's own rules and not by a decision made here —
 * the greeting, because `h1` is unconditionally coral in globals.css and always
 * is, and the selected plan-type card, because an active/selected state is one
 * of the accent's sanctioned jobs (design-guideline §3, §Badges).
 */
export default function FocusPrompt({ onSkip }: { onSkip: () => void }) {
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

  const typeCard = (type: string, title: string, blurb: string) => (
    <button
      onClick={() => handleTypeSelect(type)}
      className={`rounded-lg p-6 transition-all ${
        selectedType === type
          ? "bg-primary/10 shadow-sm ring-2 ring-primary scale-[1.02]"
          : "bg-foreground/5 hover:bg-foreground/10"
      }`}
    >
      <h3 className="mb-2 text-xl font-medium">{title}</h3>
      <p className="text-base opacity-80">{blurb}</p>
    </button>
  );

  return (
    <main className="min-h-screen p-8">
      <div className="mx-auto max-w-7xl space-y-12">
        <div>
          <h1 className="mb-2">Welcome back, {user?.name || "there"}.</h1>
          <p className="opacity-80">Pick up where you left off.</p>
        </div>

        <div className="flex min-h-[50vh] flex-col items-center justify-center">
          <h2 className="mb-12 text-center text-3xl font-medium">
            What&apos;s your focus today?
          </h2>

          <div className="mb-12 grid w-full max-w-2xl grid-cols-1 gap-4 md:grid-cols-2">
            {typeCard("project", "Project", "Something you're building")}
            {typeCard("learning", "Learning", "Something you're learning")}
          </div>
          {typeError && (
            <p className="-mt-8 mb-4 text-base text-destructive">
              Pick project or learning first
            </p>
          )}

          <div className="mb-6 w-full max-w-2xl">
            <input
              type="text"
              value={focusText}
              onChange={(e) => setFocusText(e.target.value)}
              placeholder="e.g. building a movie app, or learning microservices"
              className="w-full border-b border-foreground/20 bg-transparent px-4 py-3 text-xl text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-primary/60"
            />
          </div>

          {/* Start button — always occupies space so nothing jumps when it appears */}
          <button
            onClick={handleStart}
            disabled={!focusText.trim()}
            aria-hidden={!focusText.trim()}
            tabIndex={focusText.trim() ? 0 : -1}
            className={`w-full max-w-2xl rounded-md bg-primary py-4 text-lg font-semibold text-primary-foreground transition-all duration-300 hover:bg-primary/90 hover:shadow-lg hover:shadow-primary/30 active:scale-[0.98] ${
              focusText.trim()
                ? "pointer-events-auto opacity-100"
                : "pointer-events-none opacity-0"
            }`}
          >
            Start this plan
          </button>

          {/* Kept strategically lower, clear breathing room from the Start slot.
              A button, not a link: there is nowhere to go. */}
          <div className="mt-12 w-full max-w-2xl">
            <button
              type="button"
              onClick={onSkip}
              className="float-right rounded-md text-base text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              Skip for now
            </button>
          </div>
        </div>
      </div>
    </main>
  );
}
