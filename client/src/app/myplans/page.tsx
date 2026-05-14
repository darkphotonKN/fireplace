"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { Card } from "@/components/ui/card";
import { authFetch, searchPlans } from "@/services/api";
import { useTheme } from "@/context/ThemeContext";

const PAGE_SIZE = 10;
const SEARCH_DEBOUNCE_MS = 300;

// Interface for plan data from API
interface Plan {
  id: string;
  name: string;
  planType: string;
  focus?: string;
  description?: string;
}

// Interface for API response
interface ApiResponse {
  statusCode: number;
  message: string;
  result: Plan[];
}

export default function MyPlans() {
  // State for plans
  // - `allPlans` holds the full list returned by GET /api/plans when not searching.
  //   We paginate it client-side because that endpoint doesn't accept limit/offset.
  // - `searchResults` holds just the current page returned by GET /api/plans/search.
  const [allPlans, setAllPlans] = useState<Plan[]>([]);
  const [searchResults, setSearchResults] = useState<Plan[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [planToDelete, setPlanToDelete] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [page, setPage] = useState(0); // 0-indexed

  const { theme } = useTheme();
  const isDarkMode = theme === "dark";

  const isSearching = debouncedSearch.length > 0;

  // Derive the plans to render + whether another page exists.
  // Search mode: server-side pagination via your /api/plans/search endpoint.
  // Default mode: client-side slicing over the full list.
  const plans = isSearching
    ? searchResults
    : allPlans.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE);

  const hasMore = isSearching
    ? searchResults.length === PAGE_SIZE
    : (page + 1) * PAGE_SIZE < allPlans.length;

  // Debounce the search input and reset pagination on every new term
  useEffect(() => {
    const timer = setTimeout(() => {
      const trimmed = search.trim();
      setDebouncedSearch((prev) => {
        if (prev !== trimmed) {
          setPage(0); // new query → back to first page
        }
        return trimmed;
      });
    }, SEARCH_DEBOUNCE_MS);
    return () => clearTimeout(timer);
  }, [search]);

  // Load plans: either the full list (no search) or the paginated search results
  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      setIsLoading(true);
      setError(null);

      try {
        if (!debouncedSearch) {
          // No search term — fetch all plans (existing endpoint, no pagination)
          const response = await authFetch("http://localhost:6060/api/plans");
          if (!response.ok) {
            throw new Error(`Failed to fetch plans: ${response.statusText}`);
          }
          const data: ApiResponse = await response.json();
          if (cancelled) return;
          setAllPlans(data.result || []);
          setSearchResults([]);
        } else {
          // Search mode — use the paginated search endpoint
          const data = await searchPlans(
            debouncedSearch,
            PAGE_SIZE,
            page * PAGE_SIZE,
          );
          if (cancelled) return;
          setSearchResults(data.result || []);
        }
      } catch (err) {
        if (cancelled) return;
        console.error("Error loading plans:", err);
        setError(
          debouncedSearch
            ? "Failed to search plans"
            : "Failed to load plans",
        );
        if (debouncedSearch) {
          setSearchResults([]);
        } else {
          setAllPlans([]);
        }
      } finally {
        if (!cancelled) setIsLoading(false);
      }
    };

    load();
    return () => {
      cancelled = true;
    };
  }, [debouncedSearch, page]);

  // Clamp page if the underlying list shrinks below the current page's start
  // (e.g. after a delete takes us past the last page of the client-side slice).
  useEffect(() => {
    if (isSearching) return;
    const maxPage = Math.max(0, Math.ceil(allPlans.length / PAGE_SIZE) - 1);
    if (page > maxPage) setPage(maxPage);
  }, [allPlans, page, isSearching]);

  // Delete plan function
  const deletePlan = async (planId: string) => {
    try {
      const response = await authFetch(
        `http://localhost:6060/api/plans/${planId}`,
        {
          method: "DELETE",
        },
      );

      if (!response.ok) {
        throw new Error(`Failed to delete plan: ${response.statusText}`);
      }

      // Remove the deleted plan from whichever list we're displaying
      if (isSearching) {
        setSearchResults((prev) => prev.filter((p) => p.id !== planId));
      } else {
        setAllPlans((prev) => prev.filter((p) => p.id !== planId));
      }
      setPlanToDelete(null);
    } catch (err) {
      console.error("Error deleting plan:", err);
      setError("Failed to delete plan");
    }
  };

  return (
    <main className="min-h-screen p-8">
      <div className="max-w-7xl mx-auto space-y-12">
        {/* Header Section */}
        <div className="backdrop-blur-sm rounded-2xl p-8 shadow-lg bg-white/5 dark:bg-gray-900/10">
          <h1 className="text-4xl font-bold mb-2">My Plans</h1>
          <p className="opacity-80">
            Manage and track your development and learning plans.
          </p>
        </div>

        {/* Search Bar */}
        <div>
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search plans..."
            className="w-full px-4 py-3 rounded-xl text-base bg-white/5 dark:bg-gray-900/10 border border-foreground/10 focus:border-[rgb(247,111,83)]/50 focus:ring-1 focus:ring-[rgb(247,111,83)]/30 outline-none placeholder:text-foreground/40 transition-all backdrop-blur-sm"
          />
        </div>

        {/* Create New Plan — mirrors the sidebar's + New Plan button, right-aligned below the search bar */}
        <div className="flex justify-end -mt-6">
          <Link
            href="/create-plan"
            className="relative inline-block py-2 px-4 rounded-md text-base font-medium text-center transition-all duration-300 hover:shadow-lg hover:shadow-amber-900/20 group overflow-hidden whitespace-nowrap"
            style={{
              border: "1px solid rgb(247, 111, 83)",
              color: "rgb(247, 111, 83)",
              backgroundColor: isDarkMode
                ? "rgba(247, 111, 83, 0.05)"
                : "rgba(247, 111, 83, 0.02)",
            }}
          >
            <span className="absolute inset-0 bg-gradient-to-r from-amber-600/10 via-orange-600/10 to-amber-600/10 translate-x-full group-hover:translate-x-0 transition-transform duration-500" />
            <span className="relative z-10">+ New Plan</span>
          </Link>
        </div>

        {/* Plans Section */}
        <div className="space-y-6">
          {isLoading ? (
            <div className="text-center py-12">
              <div className="inline-block animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-orange-500"></div>
              <p className="mt-2 text-gray-500">Loading plans...</p>
            </div>
          ) : error ? (
            <div className="text-center py-12 text-red-500">{error}</div>
          ) : plans.length === 0 ? (
            isSearching ? (
              <div className="text-center py-12">
                <p className="text-gray-500">
                  No plans match &ldquo;{debouncedSearch}&rdquo;
                  {page > 0 ? " on this page" : ""}.
                </p>
                {page > 0 && (
                  <button
                    onClick={() => setPage(0)}
                    className="inline-block mt-4 px-6 py-2 rounded-md text-base font-medium transition-colors"
                    style={{
                      border: "1px solid rgb(247, 111, 83)",
                      color: "rgb(247, 111, 83)",
                    }}
                  >
                    Back to first page
                  </button>
                )}
              </div>
            ) : (
              <div className="text-center py-12">
                <p className="text-gray-500">
                  No plans found. Create your first plan!
                </p>
                <Link
                  href="/create-plan"
                  className="inline-block mt-4 px-6 py-2 rounded-md text-white font-medium transition-colors"
                  style={{ backgroundColor: "rgb(247, 111, 83)" }}
                >
                  Create New Plan
                </Link>
              </div>
            )
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
              {plans.map((plan) => (
                <div key={plan.id} className="relative group">
                  <Link href={`/plan/${plan.id}`}>
                    <Card className="backdrop-blur-sm rounded-xl shadow-xl border-0 bg-white/5 dark:bg-gray-900/10 h-full transition-all hover:shadow-2xl hover:-translate-y-1">
                      <div className="p-6">
                        <h3
                          className="text-2xl font-semibold mb-2"
                          style={{ color: "rgb(247, 111, 83)" }}
                        >
                          {plan.name}
                        </h3>
                        <div className="text-base font-medium mb-3 opacity-70">
                          {plan.planType === "project"
                            ? "Project"
                            : "Learning"}
                        </div>
                        <p className="text-base opacity-80 line-clamp-3">
                          {plan.description || "No description available"}
                        </p>
                        {plan.focus && (
                          <div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
                            <h4 className="text-base font-medium opacity-70 mb-1">
                              Focus
                            </h4>
                            <p className="text-base opacity-80 line-clamp-2">
                              {plan.focus}
                            </p>
                          </div>
                        )}
                      </div>
                    </Card>
                  </Link>
                  <button
                    onClick={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      setPlanToDelete(plan.id);
                    }}
                    className="absolute top-2 right-2 p-1.5 rounded-full opacity-0 group-hover:opacity-100 transition-opacity hover:bg-gray-100 dark:hover:bg-gray-800"
                    title="Delete plan"
                  >
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      viewBox="0 0 20 20"
                      fill="currentColor"
                      className="w-4 h-4 text-gray-500 hover:text-red-500 transition-colors"
                    >
                      <path
                        fillRule="evenodd"
                        d="M8.75 1A2.75 2.75 0 006 3.75v.443c-.795.077-1.584.176-2.365.298a.75.75 0 10.23 1.482l.149-.022.841 10.518A2.75 2.75 0 007.596 19h4.807a2.75 2.75 0 002.742-2.53l.841-10.52.149.023a.75.75 0 00.23-1.482A41.03 41.03 0 0014 4.193V3.75A2.75 2.75 0 0011.25 1h-2.5zM10 4c.84 0 1.673.025 2.5.075V3.75c0-.69-.56-1.25-1.25-1.25h-2.5c-.69 0-1.25.56-1.25 1.25v.325C8.327 4.025 9.16 4 10 4zM8.58 7.72a.75.75 0 00-1.5.06l.3 7.5a.75.75 0 101.5-.06l-.3-7.5zm4.34.06a.75.75 0 10-1.5-.06l-.3 7.5a.75.75 0 101.5.06l.3-7.5z"
                        clipRule="evenodd"
                      />
                    </svg>
                  </button>
                </div>
              ))}
            </div>
          )}

          {/* Pagination — always visible when there's anything to paginate.
              Search mode: uses /api/plans/search limit+offset.
              Default mode: paginates the full /api/plans response client-side. */}
          {!isLoading && !error && (page > 0 || hasMore || plans.length > 0) && (
            <div className="flex items-center justify-center gap-3 pt-4">
              <button
                onClick={() => setPage((p) => Math.max(0, p - 1))}
                disabled={page === 0}
                className="px-4 py-2 rounded-md text-base font-medium transition-colors bg-white/5 dark:bg-gray-900/10 border border-foreground/10 hover:bg-white/10 disabled:opacity-30 disabled:cursor-not-allowed disabled:hover:bg-white/5"
              >
                ← Previous
              </button>
              <span className="text-base opacity-70 px-2">Page {page + 1}</span>
              <button
                onClick={() => setPage((p) => p + 1)}
                disabled={!hasMore}
                className="px-4 py-2 rounded-md text-base font-medium transition-colors bg-white/5 dark:bg-gray-900/10 border border-foreground/10 hover:bg-white/10 disabled:opacity-30 disabled:cursor-not-allowed disabled:hover:bg-white/5"
              >
                Next →
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Delete Confirmation Modal */}
      {planToDelete && (
        <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-sm w-full mx-4">
            <h3 className="text-lg font-semibold mb-4">Delete Plan</h3>
            <p className="text-gray-600 dark:text-gray-300 mb-6">
              Are you sure you want to delete this plan? This action cannot be
              undone.
            </p>
            <div className="flex justify-end space-x-3">
              <button
                onClick={() => setPlanToDelete(null)}
                className="px-4 py-2 rounded-md bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => deletePlan(planToDelete)}
                className="px-4 py-2 rounded-md text-white font-medium transition-colors"
                style={{ backgroundColor: "rgb(247, 111, 83)" }}
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      )}
    </main>
  );
}
