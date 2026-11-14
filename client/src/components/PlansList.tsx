'use client';

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react';
import { listPlans } from '@/api/plans';

export interface PanelPlan {
  id: string;
  name: string;
  planType: string;
  focus?: string;
  description?: string;
  relation?: 'owner' | 'member';
}

export type PlanFilter = 'learning' | 'project';

interface PlansList {
  plans: PanelPlan[];
  isLoading: boolean;
  error: string | null;
  activeFilter: PlanFilter;
  setActiveFilter: (filter: PlanFilter) => void;
  /** Fetch the list if no panel has yet; later calls are no-ops. */
  load: () => void;
}

const PlansListContext = createContext<PlansList | null>(null);

/**
 * The plan list and the Projects / Learning choice behind the plans panel,
 * held once for the shell (FS-49CQ2 R4). The dock and the narrow-width drawer
 * both read it, so a drawer that mounts on open shows the list at once and
 * keeps the filter, and the two never fetch twice. It fetches when the first
 * panel shows, not on mount, and lives as long as the shell does, as the
 * dock's own list used to (FS-S6WVY).
 */
export function PlansListProvider({ children }: { children: ReactNode }) {
  const [plans, setPlans] = useState<PanelPlan[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeFilter, setActiveFilter] = useState<PlanFilter>('project');
  const requested = useRef(false);

  const load = useCallback(() => {
    if (requested.current) return;
    requested.current = true;

    (async () => {
      try {
        // Bare list, not data.result — this endpoint is serialized (FS-0004).
        setPlans(await listPlans());
      } catch (err) {
        console.error('Error fetching plans:', err);
        setError('Failed to load plans');
      } finally {
        setIsLoading(false);
      }
    })();
  }, []);

  const value = useMemo(
    () => ({ plans, isLoading, error, activeFilter, setActiveFilter, load }),
    [plans, isLoading, error, activeFilter, load],
  );

  return <PlansListContext.Provider value={value}>{children}</PlansListContext.Provider>;
}

/** The shell's plan list, fetched on first use. Needs a `PlansListProvider` above. */
export function usePlansList(): Omit<PlansList, 'load'> {
  const list = useContext(PlansListContext);
  if (!list) throw new Error('usePlansList needs a PlansListProvider above it');

  const { load } = list;
  useEffect(load, [load]);

  return list;
}
