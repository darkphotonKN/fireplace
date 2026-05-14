'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { authFetch } from '@/services/api';
import { useSidebar } from './LayoutWrapper';
import { useTheme } from '@/context/ThemeContext';

interface Plan {
  id: string;
  name: string;
  planType: string;
  focus?: string;
  description?: string;
}

interface ApiResponse {
  statusCode: number;
  message: string;
  result: Plan[];
}

type PlanFilter = 'learning' | 'project';

export default function Sidebar() {
  const pathname = usePathname();
  const planId = pathname?.split('/plan/')?.[1] || '';
  const { isCollapsed, setIsCollapsed } = useSidebar();

  const { theme } = useTheme();
  const isDarkMode = theme === 'dark';
  const [isHovered, setIsHovered] = useState(false);

  const [plans, setPlans] = useState<Plan[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeFilter, setActiveFilter] = useState<PlanFilter>('project');

  useEffect(() => {
    const fetchPlans = async () => {
      setIsLoading(true);
      setError(null);

      try {
        const response = await authFetch('http://localhost:6060/api/plans');

        if (!response.ok) {
          throw new Error(`Failed to fetch plans: ${response.statusText}`);
        }

        const data: ApiResponse = await response.json();
        setPlans(data.result || []);
      } catch (err) {
        console.error('Error fetching plans:', err);
        setError('Failed to load plans');
        setPlans([]);
      } finally {
        setIsLoading(false);
      }
    };

    fetchPlans();
  }, []);

  const filteredPlans = plans
    .filter((plan) => plan.planType === activeFilter)
    .map((plan) => ({
      name: plan.name,
      href: `/plan/${plan.id}`,
      isActive: plan.id === planId,
    }));

  return (
    <div
      className="fixed left-0 top-16 h-[calc(100vh-4rem)] z-20"
      onMouseEnter={() => {
        if (isCollapsed) {
          setIsHovered(true);
          localStorage.setItem('lastSidebarOpen', new Date().toISOString());
        }
      }}
      onMouseLeave={() => isCollapsed ? setIsHovered(false) : null}
    >
      {/* Collapsed edge indicator — subtle tab on the left */}
      {isCollapsed && !isHovered && (
        <div
          className="absolute left-0 top-1/2 -translate-y-1/2 w-5 h-16 flex items-center justify-center cursor-pointer rounded-r-md transition-all duration-300 opacity-40 hover:opacity-80"
          style={{
            backgroundColor: isDarkMode ? '#2a2a2a' : '#ddd8c8',
          }}
          onClick={() => {
            setIsCollapsed(false);
            localStorage.setItem('lastSidebarOpen', new Date().toISOString());
          }}
          title="Open plans panel"
        >
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" className="w-3 h-3 text-foreground/60">
            <path fillRule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clipRule="evenodd" />
          </svg>
        </div>
      )}

      {/* Hover trigger area when collapsed */}
      {isCollapsed && <div className="absolute left-0 top-0 h-full w-[50px]" />}

      <aside
        className={`h-full pt-4 px-4 overflow-y-auto transition-all duration-300 relative flex flex-col ${
          isCollapsed
            ? (isHovered ? 'translate-x-0 opacity-100' : '-translate-x-full opacity-0')
            : 'translate-x-0 opacity-100'
        }`}
        style={{
          backgroundColor: isDarkMode ? '#171717' : 'rgb(232, 230, 217)',
          width: '16rem',
        }}
      >
        {/* Header: toggle + collapse button */}
        <div className="flex items-center gap-2 mt-1 mb-4">
          <div className="flex flex-1 rounded-md overflow-hidden border border-foreground/10">
          <button
            onClick={() => setActiveFilter('project')}
            className={`flex-1 py-1.5 text-xs font-medium transition-colors ${
              activeFilter === 'project'
                ? 'bg-[rgb(247,111,83)] text-white'
                : 'text-foreground/50 hover:text-foreground/80 hover:bg-foreground/5'
            }`}
          >
            Projects
          </button>
          <button
            onClick={() => setActiveFilter('learning')}
            className={`flex-1 py-1.5 text-xs font-medium transition-colors ${
              activeFilter === 'learning'
                ? 'bg-[rgb(247,111,83)] text-white'
                : 'text-foreground/50 hover:text-foreground/80 hover:bg-foreground/5'
            }`}
          >
            Learning
          </button>
          </div>
          <button
            onClick={() => {
              const next = !isCollapsed;
              setIsCollapsed(next);
              if (!next) {
                localStorage.setItem('lastSidebarOpen', new Date().toISOString());
              }
            }}
            className="shrink-0 p-1.5 rounded-md hover:bg-black/10 dark:hover:bg-white/10 transition-colors"
            title={isCollapsed ? 'Pin sidebar' : 'Hide sidebar'}
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 20 20"
              fill="currentColor"
              className={`w-4 h-4 transition-transform text-foreground/50 ${isCollapsed ? 'rotate-180' : ''}`}
            >
              <path
                fillRule="evenodd"
                d="M12.79 5.23a.75.75 0 01-.02 1.06L8.832 10l3.938 3.71a.75.75 0 11-1.04 1.08l-4.5-4.25a.75.75 0 010-1.08l4.5-4.25a.75.75 0 011.06.02z"
                clipRule="evenodd"
              />
            </svg>
          </button>
        </div>

        {/* Plan list */}
        <nav className="flex-1 overflow-y-auto">
          <ul className="space-y-1">
            {isLoading ? (
              <li className="text-base opacity-50 px-3 py-2">Loading...</li>
            ) : error ? (
              <li className="text-base opacity-50 px-3 py-2">{error}</li>
            ) : filteredPlans.length === 0 ? (
              <li className="text-base text-foreground/40 px-3 py-2">
                No {activeFilter === 'project' ? 'projects' : 'learning plans'} yet
              </li>
            ) : (
              filteredPlans.map((item) => (
                <li key={item.href} className="group">
                  <Link
                    href={item.href}
                    className={`relative flex px-3 py-2 text-base rounded-md transition-all duration-300 cursor-pointer ${
                      item.isActive ? 'font-medium' : ''
                    }`}
                    style={{
                      backgroundColor: item.isActive ? 'rgba(247, 111, 83, 0.1)' : 'transparent',
                      color: item.isActive ? 'rgb(247, 111, 83)' : 'inherit',
                    }}
                  >
                    {/* Hover gradient effect */}
                    <span className={`absolute inset-0 rounded-md transition-all duration-300 opacity-0 group-hover:opacity-100 ${
                      !item.isActive ? 'bg-gradient-to-r from-transparent via-amber-900/10 to-transparent' : ''
                    }`} />

                    {/* Warm glow effect on hover */}
                    <span className={`absolute -inset-1 rounded-lg blur-md transition-all duration-500 opacity-0 group-hover:opacity-30 ${
                      !item.isActive ? 'bg-gradient-to-r from-amber-600/20 via-orange-600/20 to-amber-600/20' : ''
                    }`} />

                    {/* Left border indicator */}
                    <span className={`absolute left-0 top-1/2 -translate-y-1/2 w-0.5 h-0 bg-amber-500 transition-all duration-300 ${
                      item.isActive ? 'h-full' : 'group-hover:h-3/4'
                    }`} />

                    {/* Text content */}
                    <span className="relative z-10 transition-all duration-300 group-hover:translate-x-1">
                      {item.name}
                    </span>
                  </Link>
                </li>
              ))
            )}
          </ul>
        </nav>

        {/* Create New Plan button — pinned to bottom */}
        <div className="pt-4 pb-4">
          <Link
            href="/create-plan"
            className="relative block w-full py-2 px-4 rounded-md text-base font-medium text-center transition-all duration-300 hover:shadow-lg hover:shadow-amber-900/20 group overflow-hidden"
            style={{
              border: '1px solid rgb(247, 111, 83)',
              color: 'rgb(247, 111, 83)',
              backgroundColor: isDarkMode
                ? 'rgba(247, 111, 83, 0.05)'
                : 'rgba(247, 111, 83, 0.02)',
            }}
          >
            <span className="absolute inset-0 bg-gradient-to-r from-amber-600/10 via-orange-600/10 to-amber-600/10 translate-x-full group-hover:translate-x-0 transition-transform duration-500" />
            <span className="relative z-10">+ New Plan</span>
          </Link>
        </div>
      </aside>
    </div>
  );
}
