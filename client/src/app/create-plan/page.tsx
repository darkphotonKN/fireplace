'use client';

import { useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { Card } from '@/components/ui/card';
import { createPlan } from '@/api/plans';
import { toast } from '@/components/ui/use-toast';

// Define plan types for the dropdown
const PLAN_TYPES = [
  { value: 'project', label: 'Project' },
  { value: 'learning', label: 'Learning' },
];

// Interface for API response
interface ApiResponse {
  statusCode: number;
  message: string;
  result: {
    id: string;
    name: string;
    focus: string;
    description: string;
    planType: string;
  };
}

export default function CreatePlan() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  // Form state — pre-fill from query params if coming from dashboard
  const [formData, setFormData] = useState({
    name: searchParams.get('name') || '',
    focus: searchParams.get('focus') || '',
    description: '',
    planType: searchParams.get('planType') || 'project',
  });

  const prefilled = !!searchParams.get('name');

  // Handle form input changes
  const handleChange = (
    e: React.ChangeEvent<
      HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement
    >
  ) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));
  };

  // Handle form submission
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    setError(null);

    try {
      // Bare resource, not data.result — this endpoint is serialized (FS-0004).
      // The hardcoded http://localhost:6060 goes with it: the generated client
      // takes its base URL from config.
      const newPlan = await createPlan(formData);
      const newPlanId = newPlan.id;

      setSuccess(true);
      toast({ title: 'Plan created' });

      // Reset form
      setFormData({
        name: '',
        focus: '',
        description: '',
        planType: 'project',
      });

      // Redirect to home page with the new plan_id
      setTimeout(() => {
        router.push(`/plan/${newPlanId}`);
      }, 1500);
    } catch (err) {
      console.error('Error creating plan:', err);
      setError(
        err instanceof Error
          ? err.message
          : 'Failed to create plan. Please try again.'
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <main className="min-h-screen p-8">
      <div className="max-w-7xl mx-auto flex items-center justify-center h-[calc(100vh-8rem)]">
        <Card className="w-full max-w-md backdrop-blur-sm shadow-lg border-0 bg-white/5 dark:bg-gray-900/10">
          <div className="p-8">
            {/* Back button — uses router.back() so it returns to wherever the
                user came from (My Plans, dashboard, etc.). Ghost styling so it
                doesn't compete with the form; brand-colour on hover. */}
            <button
              type="button"
              onClick={() => router.back()}
              className="group inline-flex items-center gap-1.5 text-sm text-gray-400 hover:text-[rgb(247,111,83)] transition-colors mb-4 -ml-1 px-1 py-1 rounded-md focus:outline-none focus-visible:ring-1 focus-visible:ring-[rgb(247,111,83)]/50"
              aria-label="Go back"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 20 20"
                fill="currentColor"
                className="w-4 h-4 transition-transform group-hover:-translate-x-0.5"
                aria-hidden="true"
              >
                <path
                  fillRule="evenodd"
                  d="M12.79 5.23a.75.75 0 01-.02 1.06L8.832 10l3.938 3.71a.75.75 0 11-1.04 1.08l-4.5-4.25a.75.75 0 010-1.08l4.5-4.25a.75.75 0 011.06.02z"
                  clipRule="evenodd"
                />
              </svg>
              Back
            </button>

            <h1 className="text-2xl font-bold mb-6 text-center">
              Create New Plan
            </h1>

            {error && (
              <div className="mb-4 p-3 bg-red-500/10 text-red-400 text-base rounded-md">
                {error}
              </div>
            )}

            {success && (
              <div className="mb-4 p-3 bg-green-500/10 text-green-400 text-base rounded-md">
                Plan created successfully! Redirecting to your new plan...
              </div>
            )}

            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label
                  htmlFor="name"
                  className="block text-base font-medium opacity-80 mb-1"
                >
                  Plan Name
                </label>
                <input
                  type="text"
                  id="name"
                  name="name"
                  value={formData.name}
                  onChange={handleChange}
                  placeholder="NextJS Portfolio Website"
                  required
                  className="w-full px-4 py-2 border rounded-md text-base bg-white/5 border-gray-700 placeholder-gray-500"
                  disabled={isSubmitting || success}
                />
              </div>

              <div>
                <label
                  htmlFor="focus"
                  className="block text-base font-medium opacity-80 mb-1"
                >
                  Focus
                </label>
                <input
                  type="text"
                  id="focus"
                  name="focus"
                  value={formData.focus}
                  onChange={handleChange}
                  placeholder="Building a modern portfolio website using NextJS..."
                  required
                  className="w-full px-4 py-2 border rounded-md text-base bg-white/5 border-gray-700 placeholder-gray-500"
                  disabled={isSubmitting || success}
                />
              </div>

              <div>
                <label
                  htmlFor="description"
                  className="block text-base font-medium opacity-80 mb-1"
                >
                  Description
                </label>
                <textarea
                  id="description"
                  name="description"
                  value={formData.description}
                  onChange={handleChange}
                  placeholder={prefilled ? "Add a quick description to get started..." : "A personal portfolio site with sections for..."}
                  required
                  rows={3}
                  className="w-full px-4 py-2 border rounded-md text-base bg-white/5 border-gray-700 placeholder-gray-500"
                  disabled={isSubmitting || success}
                />
              </div>

              <div>
                <label
                  htmlFor="planType"
                  className="block text-base font-medium opacity-80 mb-1"
                >
                  Plan Type
                </label>
                <select
                  id="planType"
                  name="planType"
                  value={formData.planType}
                  onChange={handleChange}
                  required
                  className="w-full px-4 py-2 border rounded-md text-base bg-white/5 border-gray-700"
                  disabled={isSubmitting || success}
                >
                  {PLAN_TYPES.map((type) => (
                    <option key={type.value} value={type.value}>
                      {type.label}
                    </option>
                  ))}
                </select>
              </div>

              <div className="pt-2">
                <button
                  type="submit"
                  disabled={isSubmitting || success}
                  className="w-full px-4 py-2 rounded-md text-base text-white font-medium transition-colors"
                  style={{
                    backgroundColor:
                      isSubmitting || success
                        ? 'rgba(247, 111, 83, 0.7)'
                        : 'rgb(247, 111, 83)',
                  }}
                >
                  {isSubmitting
                    ? 'Creating...'
                    : success
                    ? 'Created!'
                    : 'Create Plan'}
                </button>
              </div>
            </form>
          </div>
        </Card>
      </div>
    </main>
  );
}
