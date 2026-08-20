"use client";

import { Card } from "@/components/ui/card";
import Todo from "@/components/Todo";
import { NotesContainer } from "@/components/notes/NotesContainer";
import { CalendarCard } from "@/components/calendar/CalendarCard";
import { getSuggestedVideos, type VideoSuggestion } from "@/api/insights";
import { useEffect, useState, use } from "react";
import { PlanDetailData } from "@/services/api";
import { getPlan } from "@/api/plans";


// Extract video ID from YouTube URL
const getYouTubeThumbnail = (url: string) => {
  const videoId = url.match(
    /(?:youtube\.com\/(?:[^\/]+\/.+\/|(?:v|e(?:mbed)?)\/|.*[?&]v=)|youtu\.be\/)([^"&?\/\s]{11})/,
  )?.[1];
  return videoId ? `https://img.youtube.com/vi/${videoId}/mqdefault.jpg` : null;
};

export default function PlanDetail({
  params,
}: {
  params: Promise<{ planId: string }>;
}) {
  const { planId } = use(params);
  const [plan, setPlan] = useState<PlanDetailData | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState("");
  const [loadingVideos, setLoadingVideos] = useState(false);
  const [videoSuggestions, setVideoSuggestions] = useState<VideoSuggestion[]>(
    [],
  );
  // The plan page used to also fetch daily + longterm checklists here in
  // parallel and stash the merged result in a `checklistItems` state — but
  // that state was never read, and the two extra fetches piled onto the
  // gateway alongside the per-Todo fetches, causing the request storm seen
  // in the network panel. Stripped: each <Todo> owns its own data, the page
  // just owns the plan metadata.
  useEffect(() => {
    async function loadPlanData() {
      if (!planId) return;

      setIsLoading(true);
      setError("");

      try {
        // Bare resource, not planResponse.result — serialized (FS-0004).
        // A failure now throws rather than returning a body with a message,
        // so the catch below is the only error path.
        setPlan(await getPlan(planId));
      } catch (error) {
        console.error("Error loading plan:", error);
        setError("Failed to load plan data");
      } finally {
        setIsLoading(false);
      }
    }

    loadPlanData();
  }, [planId]);

  useEffect(() => {
    async function loadVideoSuggestions() {
      if (!planId) return;

      setLoadingVideos(true);
      setVideoSuggestions([]);

      try {
        setVideoSuggestions(await getSuggestedVideos(planId));
      } catch (error) {
        console.error("Error fetching video suggestions:", error);
        setError("Failed to load video suggestions");
      } finally {
        setLoadingVideos(false);
      }
    }

    loadVideoSuggestions();
  }, [planId]);

  return (
    <main className="min-h-screen p-8">
      <div className="relative max-w-7xl mx-auto space-y-8">
        {/* Title Section */}
        <div className="backdrop-blur-sm rounded-2xl p-8 shadow-lg bg-white/5 dark:bg-gray-900/10">
          <h1 className="text-4xl font-bold mb-2">
            {isLoading ? "Loading..." : plan?.name || "Plan Details"}
          </h1>
          <p className="text-base opacity-80">
            {isLoading
              ? "..."
              : plan?.description || "Let's continue your development journey."}
          </p>

          <div className="absolute bottom-[20px] right-[20px] z-10">
            <div className="relative group">
              <button
                className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
                aria-label="Information about daily suggestions"
              >
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  viewBox="0 0 20 20"
                  fill="currentColor"
                  className="w-4 h-4"
                >
                  <path
                    fillRule="evenodd"
                    d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a.75.75 0 000 1.5h.253a.25.25 0 01.244.304l-.459 2.066A1.75 1.75 0 0010.747 15H11a.75.75 0 000-1.5h-.253a.25.25 0 01-.244-.304l.459-2.066A1.75 1.75 0 009.253 9H9z"
                    clipRule="evenodd"
                  />
                </svg>
              </button>
              <div className="absolute left-1/2 -translate-x-1/2 top-full mt-2 w-64 opacity-0 group-hover:opacity-100 transition-opacity duration-200 pointer-events-none z-[9999]">
                <div className="bg-gray-900 border border-gray-700 rounded-lg p-3 shadow-xl">
                  <p className="text-xs text-white mb-1">{plan?.focus}</p>
                  <p className="text-xs text-gray-400">
                    Your focus is one of the primary components that drive the
                    insights and suggestions provided for your plan.
                  </p>
                </div>
                <div className="absolute left-1/2 -translate-x-1/2 -top-1 w-2 h-2 bg-gray-900 border-l border-t border-gray-700 transform -rotate-45"></div>
              </div>
            </div>
          </div>
        </div>

        {/* Main Grid - Flexible Layout */}
        <div className="grid grid-cols-1 gap-6">
          {/* Daily — warm "reminder" strip pinned to the top of the stack. The
              accent border + tint set it apart from the neutral core blocks
              below; Todo renders its own "Daily" header internally so no extra
              title is needed here. */}
          <Card className="relative overflow-hidden backdrop-blur-sm shadow-sm border-0 border-l-4 border-l-amber-500/50 bg-gradient-to-br from-amber-500/[0.07] to-orange-500/[0.02] ring-1 ring-amber-500/10">
            <div
              aria-hidden
              className="pointer-events-none absolute -top-10 -right-10 w-40 h-40 rounded-full bg-amber-400/10 blur-3xl"
            />
            <div className="relative p-6">
              <p className="mb-3 text-xs font-medium uppercase tracking-wide text-amber-500/80">
                Today&apos;s reminders · resets daily
              </p>
              <Todo fixedTaskType="daily" dailyAIOnly />
            </div>
          </Card>

          {/* PRIMARY — Items (longterm) take the full width as the main focus.
              The dedicated Note block is hidden in this iteration; NotesContainer
              import is preserved for a future feature. */}
          <Card className="backdrop-blur-sm shadow-sm border-0">
            <div className="p-6">
              <Todo fixedTaskType="longterm" enableTypeFilter />
            </div>
          </Card>

          {/* Core blocks — Calendar (full width) */}
          <CalendarCard planId={planId} />

          {/* Recommended Videos */}
          <Card className="backdrop-blur-sm shadow-sm border-0">
            <h2 className="text-xl font-semibold p-6 pb-4">
              Recommended Videos
            </h2>
            <div className="space-y-4 p-6 pt-0">
              {loadingVideos ? (
                <div className="py-4 text-center">
                  <p className="text-gray-500 text-sm">
                    Loading video suggestions...
                  </p>
                </div>
              ) : videoSuggestions.length > 0 ? (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {videoSuggestions.map((video, index) => (
                    <a
                      key={index}
                      href={video.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="group bg-white/5 backdrop-blur-sm border border-white/10 rounded-lg overflow-hidden hover:bg-white/10 transition-all duration-200"
                    >
                      <div className="relative aspect-video">
                        <img
                          src={getYouTubeThumbnail(video.url) || ""}
                          alt={video.title}
                          className="w-full h-full object-cover"
                        />
                        <div className="absolute inset-0 bg-black/20 group-hover:bg-black/30 transition-colors flex items-center justify-center">
                          <svg
                            xmlns="http://www.w3.org/2000/svg"
                            viewBox="0 0 24 24"
                            fill="currentColor"
                            className="w-8 h-8 text-white opacity-80 group-hover:opacity-100 transition-opacity"
                          >
                            <path
                              fillRule="evenodd"
                              d="M4.5 5.653c0-1.426 1.529-2.33 2.779-1.643l11.54 6.348c1.295.712 1.295 2.573 0 3.285L7.28 19.991c-1.25.687-2.779-.217-2.779-1.643V5.653z"
                              clipRule="evenodd"
                            />
                          </svg>
                        </div>
                      </div>
                      <div className="p-3">
                        <h4 className="text-sm font-medium mb-1 line-clamp-2">
                          {video.title}
                        </h4>
                        <p className="text-xs text-gray-400 line-clamp-2">
                          {video.description}
                        </p>
                      </div>
                    </a>
                  ))}
                </div>
              ) : (
                <div className="py-4 text-center">
                  <p className="text-gray-500 text-sm">
                    No video suggestions available.
                  </p>
                </div>
              )}
            </div>
          </Card>
        </div>
      </div>
    </main>
  );
}
