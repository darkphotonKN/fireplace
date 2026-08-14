"use client";

import { useSearchParams } from 'next/navigation';
import { Calendar } from '@/components/calendar/Calendar';
import { useEffect, useState } from 'react';
import { PlanDetailData } from '@/services/api';
import { getPlan } from '@/api/plans';
import Link from 'next/link';
import { ArrowLeft } from 'lucide-react';

export default function CalendarPage() {
  const searchParams = useSearchParams();
  const planId = searchParams.get('planId');
  const [plan, setPlan] = useState<PlanDetailData | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (planId) {
      fetchPlanDetails();
    }
  }, [planId]);

  const fetchPlanDetails = async () => {
    if (!planId) return;

    try {
      setLoading(true);
      // Bare resource, not response.result — serialized (FS-0004).
      setPlan(await getPlan(planId));
    } catch (error) {
      console.error('Failed to fetch plan details:', error);
    } finally {
      setLoading(false);
    }
  };

  if (!planId) {
    return (
      <main className="min-h-screen p-8">
        <div className="max-w-7xl mx-auto">
          <div className="backdrop-blur-sm rounded-2xl p-8 shadow-lg bg-white/5 dark:bg-gray-900/10">
            <h1 className="text-3xl font-bold mb-4">Calendar</h1>
            <p className="text-gray-500">
              Please select a plan to view its calendar.
            </p>
            <Link
              href="/myplans"
              className="inline-flex items-center gap-2 mt-4 text-amber-400 hover:text-amber-300 transition-colors"
            >
              <ArrowLeft className="w-4 h-4" />
              Go to My Plans
            </Link>
          </div>
        </div>
      </main>
    );
  }

  return (
    <main className="min-h-screen p-8">
      <div className="max-w-7xl mx-auto space-y-6">
        {/* Header */}
        <div className="backdrop-blur-sm rounded-2xl p-8 shadow-lg bg-white/5 dark:bg-gray-900/10">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold mb-2">
                {loading ? 'Loading...' : plan?.name ? `${plan.name} - Calendar` : 'Calendar'}
              </h1>
              <p className="text-gray-500">
                View and manage your scheduled tasks
              </p>
            </div>
            <Link
              href={`/plan/${planId}`}
              className="inline-flex items-center gap-2 text-amber-400 hover:text-amber-300 transition-colors"
            >
              <ArrowLeft className="w-4 h-4" />
              Back to Plan
            </Link>
          </div>
        </div>

        {/* Calendar */}
        <Calendar planId={planId} />
      </div>
    </main>
  );
}