'use client';

import { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import { Checkbox } from '@/components/ui/checkbox';
import {
  fetchChecklist,
  createChecklistItem,
  updateChecklistItem,
  deleteChecklistItem,
  scheduleChecklistItem,
  ChecklistItem,
  scope,
  ScopeEnum,
  archiveChecklistItem,
  fetchArchivedChecklist,
  ChecklistResponse,
} from '@/services/api';
import { getChecklistSuggestion, getDailyInsights } from '@/api/insights';
import { getPlan, toggleDailyReset } from '@/api/plans';
import { useParams } from 'next/navigation';
import DatePicker from 'react-datepicker';
import 'react-datepicker/dist/react-datepicker.css';
import { useRouter } from 'next/navigation';
import { useToast } from '@/components/ui/use-toast';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Card } from '@/components/ui/card';
import { Progress } from '@/components/ui/progress';
import { Badge } from '@/components/ui/badge';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Separator } from '@/components/ui/separator';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { cn } from '@/lib/utils';
import {
  collapsedCount,
  findAddBarTarget,
  groupChecklistItems,
  resolveAddBarTarget,
  visibleRows,
} from '@/lib/nesting';
import { clampPage, pageCount, pageSlice } from '@/lib/paging';
import { loadCollapsedIds, saveCollapsedIds } from '@/lib/collapsedGroups';
import { format } from 'date-fns';
import {
  CalendarIcon,
  CheckCircle2,
  Clock,
  Plus,
  Trash2,
  XCircle,
  ChevronDown,
  ChevronUp,
  ChevronLeft,
  ChevronRight,
  ArrowUpDown,
  Info,
  CheckSquare,
  FileText,
  ChevronsDownUp,
  ChevronsUpDown,
} from 'lucide-react';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { Calendar } from '@/components/ui/calendar';

interface Plan {
  id: string;
  name: string;
  planType: string;
  focus?: string;
  description?: string;
  dailyReset: boolean;
}

interface VideoSuggestion {
  title: string;
  url: string;
  source: string;
  type: string;
  description: string;
}

// Previous / next share their quiet register with the header's collapse-all
// toggle (FS-0008 R5); a disabled one only dims, since it can't be hovered.
const pageButtonClass =
  'flex items-center transition-colors hover:text-primary focus-visible:text-primary disabled:pointer-events-none disabled:opacity-30';

interface TodoProps {
  /** When set, locks taskType to this value and hides the Daily/Long-term tab buttons. */
  fixedTaskType?: 'daily' | 'longterm';
  /** Surfaces the All | Notes | Checklist filter tabs over the list (longterm only). */
  enableTypeFilter?: boolean;
  /** Hides the manual "Add" form on the daily side; AI suggestions remain. */
  dailyAIOnly?: boolean;
}

type ListTypeFilter = 'all' | 'note' | 'task';

export default function Todo({
  fixedTaskType,
  enableTypeFilter = false,
  dailyAIOnly = false,
}: TodoProps = {}) {
  const params = useParams();
  const planId = params?.planId as string;
  const { toast } = useToast();

  // First-time hint for Tab-to-nest, rate-limited to once per 24h.
  const TAB_HINT_KEY = 'tabHintLastShown';
  const TAB_HINT_TTL_MS = 24 * 60 * 60 * 1000;
  const maybeShowTabHint = () => {
    if (typeof window === 'undefined') return;
    const last = Number(window.localStorage.getItem(TAB_HINT_KEY) ?? 0);
    if (Date.now() - last < TAB_HINT_TTL_MS) return;
    window.localStorage.setItem(TAB_HINT_KEY, String(Date.now()));
    toast({
      title: 'Tip: Tab to nest',
      description:
        'Tab nests the add bar under the item above, so what you add goes in as its child. Shift+Tab brings it back to top level.',
      position: 'bottom-left',
    });
  };
  const [todos, setTodos] = useState<ChecklistItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Task type state. When fixedTaskType is supplied (the plan page uses
  // <Todo fixedTaskType="longterm" /> and <Todo fixedTaskType="daily" />),
  // it is ALWAYS the source of truth — we don't seed useState and let it
  // drift, because React preserves component state across some navigation
  // patterns and the prop change would otherwise be silently ignored.
  const [internalTaskType, setInternalTaskType] = useState<
    'daily' | 'longterm' | 'archived'
  >(fixedTaskType ?? 'daily');
  const taskType: 'daily' | 'longterm' | 'archived' =
    fixedTaskType ?? internalTaskType;
  const setTaskType = (next: 'daily' | 'longterm' | 'archived') => {
    if (!fixedTaskType) setInternalTaskType(next);
  };

  // Filter tab for All | Notes | Checklist (only used when enableTypeFilter).
  const [listTypeFilter, setListTypeFilter] = useState<ListTypeFilter>('all');

  // Which page of top-level items the list is showing (FS-0008 R3). Never
  // stored: a fresh mount starts on page 1 (R9), unlike collapse state.
  const [page, setPage] = useState(1);

  // Parent Items whose children are folded away (FS-0007 R5), remembered per
  // plan and list on this device (R6). A parent not in the set is expanded.
  const [collapsedIds, setCollapsedIds] = useState<ReadonlySet<string>>(
    () => new Set()
  );
  // Mirrors collapsedIds so several changes in one event (e.g. collapse all)
  // build on each other instead of on a stale render's set.
  const collapsedIdsRef = useRef<ReadonlySet<string>>(collapsedIds);

  // Loaded in an effect, not the initializer: there is no storage during the
  // server render. The list shows "Loading tasks…" until its fetch lands, so
  // nothing renders expanded first.
  useEffect(() => {
    if (!planId) return;
    const loaded = loadCollapsedIds(planId, taskType);
    collapsedIdsRef.current = loaded;
    setCollapsedIds(loaded);
  }, [planId, taskType]);

  // Type to use when creating the next item via the add form.
  const [newTodoType, setNewTodoType] = useState<'task' | 'note'>('task');

  const [newTodo, setNewTodo] = useState('');
  // Parent the add bar is nested under (Tab), or null at top level. Never
  // persisted, so every load starts at top level (FS-0007 R8.10).
  const [addBarParentId, setAddBarParentId] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const newTodoInputRef = useRef<HTMLInputElement>(null);

  const [editingId, setEditingId] = useState<string | null>(null);
  const [editText, setEditText] = useState('');
  const [isUpdating, setIsUpdating] = useState(false);
  const editInputRef = useRef<HTMLInputElement>(null);

  // Scheduling state
  const [schedulingId, setSchedulingId] = useState<string | null>(null);
  const [scheduleDate, setScheduleDate] = useState<Date | null>(null);
  const [isScheduling, setIsScheduling] = useState(false);

  // AI suggestion state
  const [suggestion, setSuggestion] = useState<string | null>(null);
  const [displaySuggestion, setDisplaySuggestion] = useState('');
  const [isSuggestionTyping, setIsSuggestionTyping] = useState(false);
  const [isFetchingSuggestion, setIsFetchingSuggestion] = useState(false);

  // Typing animation state
  const [isTyping, setIsTyping] = useState(false);
  const [typingIndex, setTypingIndex] = useState(0);
  const [fullText, setFullText] = useState('');
  const typingSpeed = 15; // milliseconds per character (faster)

  // New todo animation state
  const [newTodoAnimations, setNewTodoAnimations] = useState<
    Record<string, boolean>
  >({});

  // Daily insights state
  const [dailyInsights, setDailyInsights] = useState<string[]>([]);
  const [showInsights, setShowInsights] = useState(true);
  const [visibleInsights, setVisibleInsights] = useState<boolean[]>([]);

  // Archived state
  const [showArchived, setShowArchived] = useState(false);
  const [archivedTodos, setArchivedTodos] = useState<ChecklistItem[]>([]);

  // Settings state
  const [showSettings, setShowSettings] = useState(false);
  const [refreshDailyTasks, setRefreshDailyTasks] = useState(true);
  const [dailyReset, setDailyReset] = useState(false);
  const [isTogglingReset, setIsTogglingReset] = useState(false);
  const [lastToggleTime, setLastToggleTime] = useState(0);
  const TOGGLE_THROTTLE_MS = 1000; // 1 second throttle

  // Delete confirmation state
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [todoToDelete, setTodoToDelete] = useState<ChecklistItem | null>(null);

  // Load show/hide preference from localStorage on mount
  useEffect(() => {
    const savedPreference = localStorage.getItem('showDailyInsights');
    if (savedPreference !== null) {
      setShowInsights(savedPreference === 'true');
    }
  }, []);

  // Save show/hide preference to localStorage
  useEffect(() => {
    localStorage.setItem('showDailyInsights', String(showInsights));
  }, [showInsights]);

  // Handle sequential fade-up animation for insights
  useEffect(() => {
    if (!dailyInsights.length) return;

    // Initialize all insights as hidden
    setVisibleInsights(new Array(dailyInsights.length).fill(false));

    // Show each insight with a delay
    dailyInsights.forEach((_, index) => {
      setTimeout(() => {
        setVisibleInsights((prev) => {
          const newState = [...prev];
          newState[index] = true;
          return newState;
        });
      }, index * 300); // 300ms delay between each insight
    });
  }, [dailyInsights]);

  // Load refresh preference from localStorage on mount
  useEffect(() => {
    const savedPreference = localStorage.getItem('refreshDailyTasks');
    if (savedPreference !== null) {
      setRefreshDailyTasks(savedPreference === 'true');
    }
  }, []);

  // Save refresh preference to localStorage
  useEffect(() => {
    localStorage.setItem('refreshDailyTasks', String(refreshDailyTasks));
  }, [refreshDailyTasks]);

  // Fetch todos on component mount and when planId or taskType changes.
  //
  // Race-condition guard: under React StrictMode (dev double-invoke) and any
  // parent re-render that causes this effect to re-fire mid-flight, multiple
  // fetches can be in-flight at once. Without a guard, whichever resolves
  // LAST wins — including the catch-block's setTodos([]) from a transient
  // error, which silently wipes data a sibling fetch had just loaded.
  //
  // `cancelled` short-circuits any state writes from a stale effect. The
  // cleanup function flips it on next-effect-run / unmount.
  useEffect(() => {
    let cancelled = false;

    const loadTodos = async () => {
      try {
        setLoading(true);
        // Bare array, not {statusCode, message, result} — serialized (FS-0004).
        let items: ChecklistItem[];

        if (taskType === 'archived') {
          items = await fetchArchivedChecklist(planId, 'daily');
        } else {
          items = await fetchChecklist(
            planId,
            taskType as 'daily' | 'longterm'
          );
        }

        if (cancelled) return;
        setTodos(items);
        setError(null);
      } catch (error) {
        if (cancelled) return;
        console.error('Failed to fetch checklist items:', error);
        setError('Failed to load tasks. Please try again later.');
        // Intentionally do NOT setTodos([]) here. A transient gateway/gRPC
        // error shouldn't wipe data that a successful concurrent fetch may
        // already have loaded. Stale-but-correct beats empty-and-wrong.
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    loadTodos();

    // If we're in daily mode, also fetch the daily insights
    if (taskType === 'daily') {
      fetchDailyInsights();
    } else {
      // Clear insights when not in daily mode
      setDailyInsights([]);
    }

    return () => {
      cancelled = true;
    };
  }, [planId, taskType]);

  // Handle typing animation for input
  useEffect(() => {
    if (!isTyping || typingIndex >= fullText.length) return;

    const timeout = setTimeout(() => {
      setNewTodo(fullText.slice(0, typingIndex + 1));
      setTypingIndex((prevIndex) => prevIndex + 1);
    }, typingSpeed);

    return () => clearTimeout(timeout);
  }, [isTyping, typingIndex, fullText]);

  // End typing animation when complete
  useEffect(() => {
    if (isTyping && typingIndex >= fullText.length) {
      setIsTyping(false);
    }
  }, [typingIndex, fullText, isTyping]);

  // Handle typing animation for suggestion display
  useEffect(() => {
    if (!isSuggestionTyping || !suggestion) return;

    let currentIndex = 0;
    setDisplaySuggestion('');

    const interval = setInterval(() => {
      if (currentIndex < suggestion.length) {
        setDisplaySuggestion((prev) => prev + suggestion[currentIndex]);
        currentIndex++;
      } else {
        clearInterval(interval);
        setIsSuggestionTyping(false);
      }
    }, typingSpeed);

    return () => clearInterval(interval);
  }, [isSuggestionTyping, suggestion, typingSpeed]);

  // Focus on edit input when entering edit mode
  useEffect(() => {
    if (editingId && editInputRef.current) {
      editInputRef.current.focus();
    }
  }, [editingId]);

  // Toggle todo completion status
  const toggleTodo = async (id: string) => {
    // Find the todo
    const todoToToggle = todos?.find((todo) => todo.id === id);
    if (!todoToToggle) return;

    // Optimistic update
    const newDoneStatus = !todoToToggle.done;
    setTodos(
      todos?.map((todo) =>
        todo.id === id ? { ...todo, done: newDoneStatus } : todo
      )
    );

    try {
      // API update
      const response = await updateChecklistItem(
        id,
        { done: newDoneStatus },
        planId,
        taskType as 'daily' | 'longterm'
      );
    } catch (error) {
      console.error('Error toggling todo:', error);
      // Revert if exception
      setTodos(
        todos?.map((todo) =>
          todo.id === id ? { ...todo, done: todoToToggle.done } : todo
        )
      );
      setError('Failed to update task status. Please try again.');
    }
  };

  // Start scheduling a todo
  const startScheduling = (todo: ChecklistItem) => {
    setSchedulingId(todo.id);
    setScheduleDate(
      todo.scheduledTime ? new Date(todo.scheduledTime) : new Date()
    );
  };

  // Cancel scheduling
  const cancelScheduling = () => {
    setSchedulingId(null);
    setScheduleDate(null);
  };

  // Schedule a todo
  const scheduleTodo = async () => {
    if (!schedulingId || !scheduleDate) return;

    // Find the original todo
    const todoToSchedule = todos?.find((todo) => todo.id === schedulingId);
    if (!todoToSchedule) return;

    // Store the original scheduledTime
    const originalScheduledTime = todoToSchedule.scheduledTime;

    // Optimistic update
    setIsScheduling(true);
    setTodos(
      todos?.map((todo) =>
        todo.id === schedulingId
          ? { ...todo, scheduledTime: scheduleDate.toISOString() }
          : todo
      )
    );

    try {
      // API update
      const response = await scheduleChecklistItem(
        schedulingId,
        planId,
        scheduleDate,
        taskType as 'daily' | 'longterm'
      );

    } catch (error) {
      console.error('Error scheduling todo:', error);
      // Revert if exception
      setTodos(
        todos.map((todo) =>
          todo.id === schedulingId
            ? { ...todo, scheduledTime: originalScheduledTime }
            : todo
        )
      );
      setError('Failed to schedule task. Please try again.');
    } finally {
      setIsScheduling(false);
      setSchedulingId(null);
      setScheduleDate(null);
    }
  };

  const startEditing = (todo: ChecklistItem) => {
    setEditingId(todo.id);
    setEditText(todo.description);
  };

  const cancelEditing = () => {
    setEditingId(null);
    setEditText('');
  };

  // Check if scheduled time is in the past
  const isScheduledTimePast = (dateString: string) => {
    const scheduledDate = new Date(dateString);
    const now = new Date();
    return scheduledDate < now;
  };

  // Format date for display
  const formatScheduleTime = (dateString: string) => {
    const date = new Date(dateString);

    // Format: "Today at 3:00 PM" or "Tomorrow at 3:00 PM" or "Mon, Jan 1 at 3:00 PM"
    const today = new Date();
    const tomorrow = new Date(today);
    tomorrow.setDate(tomorrow.getDate() + 1);

    const isToday = date.toDateString() === today.toDateString();
    const isTomorrow = date.toDateString() === tomorrow.toDateString();

    const timeString = date.toLocaleTimeString([], {
      hour: 'numeric',
      minute: '2-digit',
    });

    if (isToday) {
      return `Today at ${timeString}`;
    } else if (isTomorrow) {
      return `Tomorrow at ${timeString}`;
    } else {
      return `${date.toLocaleDateString([], {
        weekday: 'short',
        month: 'short',
        day: 'numeric',
      })} at ${timeString}`;
    }
  };

  const handleMoveItem = async (id: string, scope: ScopeEnum) => {
    // optimisitic update
    const oldTodos = todos;
    const newTodos = todos?.filter((todo) => todo.id !== id);
    setTodos(newTodos);
    setEditingId(null);

    try {
      const res = await updateChecklistItem(id, { scope }, planId, scope);
      console.log('res:', res);
    } catch (error) {
      // undo optimistic update
      console.error(`Error moving item to ${scope}:`, error);
      setTodos(oldTodos);
      setError(`Error when attempting to move item to ${scope}.`);
    }
  };

  // Update todo description
  const updateTodoDescription = async (id: string) => {
    if (editText.trim() === '') return;

    // Find the original todo
    const originalTodo = todos?.find((todo) => todo.id === id);
    if (!originalTodo) return;

    const originalText = originalTodo.description;

    // Optimistic update
    setIsUpdating(true);

    const updatedText = editText.trim();

    setTodos(
      todos?.map((todo) =>
        todo.id === id ? { ...todo, description: updatedText } : todo
      )
    );

    try {
      // API update
      const response = await updateChecklistItem(
        id,
        {
          description: editText.trim(),
        },
        planId,
        taskType as 'daily' | 'longterm'
      );

    } catch (error) {
      console.error('Error updating todo description:', error);
      // Revert if exception
      setTodos(
        todos?.map((todo) =>
          todo.id === id ? { ...todo, description: originalText } : todo
        )
      );
      setError('Failed to update task. Please try again.');
    } finally {
      setIsUpdating(false);
      setEditingId(null);
      setEditText('');
    }
  };

  // Add a new todo
  const addTodo = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newTodo.trim() || isSubmitting || isTyping) return;

    try {
      setIsSubmitting(true);
      const newItem = await createChecklistItem(
        newTodo,
        planId,
        taskType as 'daily' | 'longterm',
        // addBarParent is only ever a real top-level task, never a note or
        // a child, so this can't ask for a nest the server would refuse.
        { type: newTodoType, parentId: addBarParent?.id }
      );
      // A child landing in a collapsed parent would be invisible; open it so
      // the arrival is seen, and remember that (R8.9). After the create, so a
      // failure leaves the parent as it was.
      if (addBarParent) setParentCollapsed(addBarParent.id, false);
      setTodos((prev) => [...prev, newItem]);
      setNewTodo('');
      // Add animation for the new todo
      setNewTodoAnimations((prev) => ({
        ...prev,
        [newItem.id]: true,
      }));
      // Remove animation after 1 second
      setTimeout(() => {
        setNewTodoAnimations((prev) => ({
          ...prev,
          [newItem.id]: false,
        }));
      }, 1000);
    } catch (error) {
      console.error('Failed to add todo:', error);
    } finally {
      setIsSubmitting(false);
      // Keep the cursor in the add bar so items can be entered back to back,
      // including when the Add button (not Enter) was used, and so a failed
      // add leaves the text ready to retry.
      newTodoInputRef.current?.focus();
    }
  };

  // Get AI suggestion for a new todo
  const getAISuggestion = async () => {
    if (isFetchingSuggestion || isTyping || isSuggestionTyping) return;

    try {
      setIsFetchingSuggestion(true);
      const suggestion = await getChecklistSuggestion(planId);
      setSuggestion(suggestion);
    } catch (error) {
      console.error('Failed to get suggestion:', error);
    } finally {
      setIsFetchingSuggestion(false);
    }
  };

  // Start typing animation for the suggestion
  const useSuggestion = () => {
    if (suggestion) {
      // Clear current input and prepare for typing animation
      setNewTodo('');
      setFullText(suggestion);
      setTypingIndex(0);
      setIsTyping(true);
      setSuggestion(null);
      setDisplaySuggestion('');
    }
  };

  console.log('@Debug todos:', todos);

  // Delete a todo
  const deleteTodo = async (todoId: string) => {
    if (!planId) return;

    setIsUpdating(true);
    try {
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/api/plans/${planId}/checklists/${todoId}`,
        {
          method: 'DELETE',
          credentials: 'include',
        }
      );

      if (!response.ok) {
        throw new Error('Failed to delete task');
      }

      // Remove the deleted todo from the state
      setTodos((prevTodos) => prevTodos?.filter((todo) => todo.id !== todoId));
    } catch (error) {
      console.error('Error deleting task:', error);
      setError('Failed to delete task');
    } finally {
      setIsUpdating(false);
    }
  };

  // Fetch daily insights from API
  const fetchDailyInsights = async () => {
    try {
      // First check if there are any long-term items
      const longTermItems = await fetchChecklist(planId, 'longterm');
      if (longTermItems.length === 0) {
        // No long-term items, so no need to fetch insights
        setDailyInsights([]);
        setVisibleInsights([]);
        return;
      }

      const response = await getDailyInsights(planId);
      if (response.length > 0) {
        setDailyInsights(response);
      } else {
        // Clear insights if API returns empty results
        setDailyInsights([]);
        setVisibleInsights([]);
      }
    } catch (error) {
      console.error('Failed to fetch daily insights:', error);
      // Clear insights on error
      setDailyInsights([]);
      setVisibleInsights([]);
    }
  };

  // Add insight as a new todo
  const addInsightAsTodo = async (description: string) => {
    try {
      // Create a temporary ID for the optimistic update
      const tempId = `insight_${Date.now()}`;

      // Create an optimistic todo item
      const tempNewItem: ChecklistItem = {
        id: tempId,
        description: description,
        done: false,
      };

      // Start animation for the new todo
      setNewTodoAnimations((prev) => ({ ...prev, [tempId]: true }));

      // Add to list with optimistic update
      setTodos((prevTodos) => [...prevTodos, tempNewItem]);

      // Actual API call
      const newItem = await createChecklistItem(
        description,
        planId,
        taskType as 'daily' | 'longterm'
      );

      // Update the item with the real ID from the API
      setTodos((prevTodos) =>
        prevTodos?.map((todo) => (todo.id === tempId ? newItem : todo))
      );

      // Remove the suggestion from the list to provide feedback that it was added
      setDailyInsights((prevInsights) =>
        prevInsights.filter((insight) => insight !== description)
      );

      // Clear errors if any
      setError(null);
    } catch (error) {
      console.error('Failed to add suggested task:', error);
      setError('Failed to add suggested task. Please try again.');

      // Create a fallback ID for the item in case of API failure
      const fallbackId = `fallback_${Date.now()}`;

      // Add as a local item even if API fails
      const fallbackItem = {
        id: fallbackId,
        description: description,
        done: false,
      };

      setTodos((prevTodos) => [...prevTodos, fallbackItem]);

      // Add animation for the fallback item
      setNewTodoAnimations((prev) => ({
        ...prev,
        [fallbackId]: true,
      }));

      // Remove the animation after delay
      setTimeout(() => {
        setNewTodoAnimations((current) => {
          const final = { ...current };
          delete final[fallbackId];
          return final;
        });
      }, 1000);
    }
  };

  // Toggle insights visibility
  const toggleInsightsVisibility = () => {
    setShowInsights((prev) => !prev);
  };

  // Archive a todo
  const archiveTodo = async (id: string) => {
    // Find the original todo before archiving it
    const todoToArchive = todos?.find((todo) => todo.id === id);
    if (!todoToArchive) return;

    // Optimistic archive
    setTodos(todos?.filter((todo) => todo.id !== id));
    setArchivedTodos([...archivedTodos, todoToArchive]);

    try {
      const response = await archiveChecklistItem(
        id,
        planId,
        taskType as 'daily' | 'longterm'
      );
    } catch (error) {
      console.error('Error archiving todo:', error);
      // Restore if exception
      setTodos((prevTodos) => [...prevTodos, todoToArchive]);
      setArchivedTodos(archivedTodos?.filter((todo) => todo.id !== id));
      setError('Failed to archive task. Please try again.');
    }
  };

  // Flip type between 'task' and 'note'.
  // Switching to 'note' optimistically clears `done` (backend enforces the same).
  const toggleTodoType = async (id: string) => {
    const todo = todos?.find((t) => t.id === id);
    if (!todo) return;
    const currentType = todo.type ?? 'task';
    const nextType = currentType === 'task' ? 'note' : 'task';
    const previous = { type: currentType, done: todo.done };

    // Optimistic update
    setTodos(
      todos?.map((t) =>
        t.id === id
          ? { ...t, type: nextType, done: nextType === 'note' ? false : t.done }
          : t
      )
    );

    try {
      const response = await updateChecklistItem(
        id,
        { type: nextType },
        planId,
        taskType as 'daily' | 'longterm'
      );
    } catch (err) {
      console.error('Error toggling item type:', err);
      setTodos((prev) =>
        prev.map((t) =>
          t.id === id ? { ...t, type: previous.type, done: previous.done } : t
        )
      );
      setError('Failed to change item type. Please try again.');
    }
  };

  // Apply the active type filter (used by the longterm "All | Notes | Checklist"
  // tab strip). For daily / archived, this is a no-op.
  const filteredTodos = useMemo(() => {
    if (!todos) return [];
    if (!enableTypeFilter || taskType !== 'longterm') return todos;
    if (listTypeFilter === 'all') return todos;
    return todos.filter((t) => (t.type ?? 'task') === listTypeFilter);
  }, [todos, enableTypeFilter, listTypeFilter, taskType]);

  // Top-level rows, each with its visible children. Children whose parent
  // isn't in the filtered set fall through as top-level (no visual indent).
  const rowGroups = useMemo(
    () => groupChecklistItems(filteredTodos),
    [filteredTodos]
  );

  // Render order: each top-level row followed by its children.
  const orderedRows = useMemo(
    () => rowGroups.flatMap((g) => [g.item, ...g.children]),
    [rowGroups]
  );

  // The add bar's target while nested, re-checked on every render so it drops
  // out once the parent is archived, converted, outdented or filtered out.
  const addBarParent = useMemo(
    () => resolveAddBarTarget(rowGroups, addBarParentId),
    [rowGroups, addBarParentId]
  );
  // Short enough for the placeholder; the aria-label carries the full text.
  const addBarParentLabel =
    addBarParent && addBarParent.description.length > 32
      ? `${addBarParent.description.slice(0, 31).trimEnd()}…`
      : addBarParent?.description;
  // The rail only reaches the add bar from the last group; a target higher
  // up still receives the child, and the input names it instead.
  const addBarRailParentId =
    addBarParent && rowGroups[rowGroups.length - 1]?.item.id === addBarParent.id
      ? addBarParent.id
      : null;

  // Forget an ineligible target rather than holding it, so it can't silently
  // re-nest the bar when, say, the type filter is switched back.
  useEffect(() => {
    if (addBarParentId && !addBarParent) setAddBarParentId(null);
  }, [addBarParentId, addBarParent]);

  // Tab nests the add bar under the parent above; Shift+Tab brings it back.
  // Both are swallowed even when nothing changes, so focus never leaves the
  // input mid-entry (R8.2–R8.8).
  const handleAddBarKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== 'Tab') return;
    e.preventDefault();
    if (e.shiftKey) {
      setAddBarParentId(null);
    } else if (!addBarParent) {
      const target = findAddBarTarget(rowGroups);
      setAddBarParentId(target?.id ?? null);
      // Joining a collapsed parent opens it, so what you type lands in view
      // and the parent's own rail is there for the bar's to meet (R8.9).
      if (target) setParentCollapsed(target.id, false);
    }
  };

  // Each row's place on a guide rail: a parent with visible children starts
  // one and its children continue it. Rows not in the map draw no rail.
  // A childless parent also starts one while the add bar is joining it.
  const guideRail = useMemo(() => {
    const roles = new Map<string, 'parent' | 'child'>();
    for (const { item, children } of rowGroups) {
      if (children.length === 0 && item.id !== addBarRailParentId) continue;
      roles.set(item.id, 'parent');
      for (const child of children) roles.set(child.id, 'child');
    }
    return roles;
  }, [rowGroups, addBarRailParentId]);

  // Pages are cut after the type filter, so what is paged is what is shown
  // (R4). The page is clamped on the way out rather than only when it changes,
  // so archiving a whole page lands on the new last page instead of an empty
  // one (R10) even before the effect below tidies the state.
  const totalPages = pageCount(rowGroups.length);
  const currentPage = clampPage(page, rowGroups.length);
  const pagedGroups = useMemo(
    () => pageSlice(rowGroups, currentPage),
    [rowGroups, currentPage]
  );

  // Keep the state honest once clamped, or a page the list has grown back into
  // would resurrect the moment the items return.
  useEffect(() => {
    if (currentPage !== page) setPage(currentPage);
  }, [currentPage, page]);

  // A different list, or a different slice of it, always starts again at the
  // top (R8). taskType covers the internal daily / long-term switcher; when
  // fixedTaskType pins it, only the filter can move.
  useEffect(() => {
    setPage(1);
  }, [taskType, listTypeFilter]);

  // What the list draws: this page's rows minus the children of collapsed
  // parents. Everything else — counts, the rail, collapse — stays over the
  // whole set (R11).
  const renderedRows = useMemo(
    () => visibleRows(pagedGroups, collapsedIds),
    [pagedGroups, collapsedIds]
  );

  // Visible children of each parent that has any. Only these parents get a
  // chevron, so a stale collapsed id on a now-childless row shows nothing.
  const childrenOf = useMemo(
    () =>
      new Map(
        rowGroups
          .filter((g) => g.children.length > 0)
          .map((g) => [g.item.id, g.children])
      ),
    [rowGroups]
  );

  const isCollapsed = (id: string) =>
    childrenOf.has(id) && collapsedIds.has(id);

  // Collapse/expand all acts on the parents on screen only, so parents the
  // type filter hides keep whatever state they were remembered with (R7.4).
  const collapsibleIds = useMemo(() => [...childrenOf.keys()], [childrenOf]);
  const anyCollapsed = collapsibleIds.some((id) => collapsedIds.has(id));

  // Parents opened this session. Only their children play the reveal, so a
  // plain page load doesn't animate every child row.
  const revealedIds = useRef(new Set<string>());

  // Collapse or expand several parents as one change, so "collapse all" is a
  // single render and a single write rather than one per parent.
  const setParentsCollapsed = (ids: readonly string[], collapsed: boolean) => {
    const prev = collapsedIdsRef.current;
    const next = new Set(prev);
    for (const id of ids) {
      if (collapsed) {
        revealedIds.current.delete(id);
        next.add(id);
      } else {
        revealedIds.current.add(id);
        next.delete(id);
      }
    }
    // One call only ever adds or only ever removes, so equal size means the
    // set is unchanged.
    if (next.size === prev.size) return;
    collapsedIdsRef.current = next;
    setCollapsedIds(next);

    // Remembered here, on the user's action, not in an effect on collapsedIds:
    // that would also fire when stored state loads, before todos arrive, and
    // prune every id. Live parents come from the unfiltered todos so a type
    // filter never prunes the parents it hides (R7.4).
    if (planId) {
      const liveParentIds = new Set(
        todos.flatMap((t) => (t.parentId ? [t.parentId] : []))
      );
      saveCollapsedIds(planId, taskType, next, liveParentIds);
    }
  };

  const setParentCollapsed = (id: string, collapsed: boolean) =>
    setParentsCollapsed([id], collapsed);

  const toggleCollapsed = (id: string) =>
    setParentCollapsed(id, !isCollapsed(id));

  // Indent a row under the nearest top-level row above it in render order.
  // We walk upward past any child rows so indenting row 3 still works after
  // row 2 has been nested under row 1 — the target is row 1.
  const indentTodo = async (id: string) => {
    const idx = orderedRows.findIndex((t) => t.id === id);
    if (idx <= 0) return; // first row, nothing above
    // Walk up to find the nearest top-level row
    let above: ChecklistItem | null = null;
    for (let i = idx - 1; i >= 0; i--) {
      if (!orderedRows[i].parentId) {
        above = orderedRows[i];
        break;
      }
    }
    if (!above) return; // nothing top-level above us
    const self = orderedRows[idx];
    if (self.parentId === above.id) return; // already nested under that row
    // pre-flight: don't try to re-parent a row that has children
    const hasChildren = todos.some((t) => t.parentId === self.id);
    if (hasChildren) return;
    // Joining a collapsed parent opens it, so the row doesn't vanish.
    setParentCollapsed(above.id, false);

    const previousParentId = self.parentId ?? null;
    // Optimistic update
    setTodos((prev) =>
      prev.map((t) => (t.id === id ? { ...t, parentId: above.id } : t))
    );

    try {
      const response = await updateChecklistItem(
        id,
        { parentId: above.id },
        planId,
        taskType as 'daily' | 'longterm'
      );
    } catch (err) {
      console.error('Error indenting item:', err);
      setTodos((prev) =>
        prev.map((t) =>
          t.id === id ? { ...t, parentId: previousParentId } : t
        )
      );
      setError('Failed to indent item. Please try again.');
    }
  };

  // Outdent a row back to top-level (parent_id := null).
  const outdentTodo = async (id: string) => {
    const self = todos?.find((t) => t.id === id);
    if (!self || !self.parentId) return; // already top-level
    const previousParentId = self.parentId;
    setTodos((prev) =>
      prev.map((t) => (t.id === id ? { ...t, parentId: null } : t))
    );

    try {
      const response = await updateChecklistItem(
        id,
        { parentId: null },
        planId,
        taskType as 'daily' | 'longterm'
      );
    } catch (err) {
      console.error('Error outdenting item:', err);
      setTodos((prev) =>
        prev.map((t) =>
          t.id === id ? { ...t, parentId: previousParentId } : t
        )
      );
      setError('Failed to outdent item. Please try again.');
    }
  };

  // Handle Tab / Shift+Tab on a focused row or edit input.
  const handleRowKeyDown = (e: React.KeyboardEvent, id: string) => {
    if (e.key !== 'Tab') return;
    e.preventDefault();
    if (e.shiftKey) {
      outdentTodo(id);
    } else {
      indentTodo(id);
    }
  };

  // Load archived todos
  const loadArchivedTodos = async () => {
    try {
      setArchivedTodos(await fetchArchivedChecklist(planId, 'daily'));
    } catch (error) {
      console.error('Failed to load archived todos:', error);
      setError('Failed to load archived tasks. Please try again.');
    }
  };

  // Load archived todos when showing archived view
  useEffect(() => {
    if (showArchived) {
      loadArchivedTodos();
    }
  }, [showArchived, planId]);

  // Delete archived todo with confirmation
  const confirmDeleteTodo = (todo: ChecklistItem) => {
    setTodoToDelete(todo);
    setShowDeleteModal(true);
  };

  const handleDeleteConfirm = async () => {
    if (!todoToDelete) return;

    try {
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/api/plans/${planId}/checklists/${todoToDelete.id}`,
        {
          method: 'DELETE',
          credentials: 'include',
        }
      );

      if (!response.ok) {
        throw new Error('Failed to delete task');
      }

      // Remove the deleted todo from the state
      setArchivedTodos((prevTodos) =>
        prevTodos.filter((todo) => todo.id !== todoToDelete.id)
      );
    } catch (error) {
      console.error('Error deleting task:', error);
      setError('Failed to delete task');
    } finally {
      setShowDeleteModal(false);
      setTodoToDelete(null);
    }
  };

  // Fetch plan metadata (dailyReset). This effect owns PLAN state only — it
  // must never touch `todos`. The taskType effect above is the single writer
  // of the checklist, and it is the only one that knows which scope this
  // instance is showing. This effect used to also fetch scope 'daily' and
  // setTodos with the result; because it does not depend on taskType, on a
  // <Todo fixedTaskType="longterm" /> instance it overwrote the long-term list
  // with daily items once its slower Promise.all settled — making a
  // just-created long-term item vanish on reload even though it had persisted.
  useEffect(() => {
    const loadPlanDetails = async () => {
      if (!planId) return;

      try {
        const planResponse = await getPlan(planId);
        // Bare resource, not planResponse.result — serialized (FS-0004).
        setDailyReset(planResponse.dailyReset);
      } catch (error) {
        console.error('Error fetching plan details:', error);
      }
    };

    loadPlanDetails();
  }, [planId]);

  // Throttled toggle handler
  const handleToggleDailyReset = useCallback(async () => {
    const now = Date.now();
    if (now - lastToggleTime < TOGGLE_THROTTLE_MS) return;

    setIsTogglingReset(true);
    setLastToggleTime(now);

    // Optimistic update
    setDailyReset((prev) => !prev);

    try {
      // Serialized: a failure THROWS rather than returning a body with a
      // statusCode to inspect, so the catch below owns the revert. The old
      // check also meant a 201/204 would have been treated as a failure.
      const plan = await toggleDailyReset(planId);
      setDailyReset(plan.dailyReset);
    } catch (error) {
      console.error('Error toggling daily reset:', error);
      setError('Failed to update daily reset setting');
    } finally {
      setIsTogglingReset(false);
    }
  }, [planId, lastToggleTime]);

  if (loading) {
    return <div className="py-4">Loading tasks...</div>;
  }

  return (
    <div className="space-y-4">
      {/* Header with Back button when in settings/archived view */}
      {(showSettings || showArchived) && (
        <button
          onClick={() => {
            if (showArchived) {
              setShowArchived(false);
            } else {
              setShowSettings(false);
            }
          }}
          className="flex items-center text-base text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 mb-4"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 20 20"
            fill="currentColor"
            className="w-4 h-4 mr-1"
          >
            <path
              fillRule="evenodd"
              d="M9.707 14.707a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414l4-4a1 1 0 011.414 1.414L7.414 9H15a1 1 0 110 2H7.414l2.293 2.293a1 1 0 010 1.414z"
              clipRule="evenodd"
            />
          </svg>
          {showArchived ? 'Back to settings' : 'Back to tasks'}
        </button>
      )}

      {/* Section Title with Task Type Switcher or Settings Title */}
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-lg font-semibold">
          {showSettings
            ? 'Settings'
            : showArchived
            ? 'Archived Tasks'
            : taskType === 'daily'
            ? 'Daily'
            : 'Items'}
        </h2>
        {!showSettings && !showArchived && (
          <div className="flex items-center gap-3">
            {enableTypeFilter && taskType === 'longterm' && (
              <div className="flex items-center gap-1 text-sm">
                {(
                  [
                    ['all', 'All'],
                    ['note', 'Notes'],
                    ['task', 'Checklist'],
                  ] as const
                ).map(([value, label]) => (
                  <button
                    key={value}
                    onClick={() => setListTypeFilter(value)}
                    className={`px-2 py-1 rounded transition-colors ${
                      listTypeFilter === value
                        ? 'bg-amber-500/20 text-amber-400'
                        : 'opacity-60 hover:opacity-100 hover:bg-white/5'
                    }`}
                    aria-pressed={listTypeFilter === value}
                  >
                    {label}
                  </button>
                ))}
              </div>
            )}
            {/* One quiet toggle for every parent on screen (R7). */}
            {collapsibleIds.length > 0 && (
              <button
                onClick={() =>
                  setParentsCollapsed(collapsibleIds, !anyCollapsed)
                }
                className="flex items-center gap-1.5 text-sm text-foreground/50 transition-colors hover:text-primary"
              >
                {anyCollapsed ? (
                  <ChevronsUpDown strokeWidth={1.75} className="h-3.5 w-3.5" />
                ) : (
                  <ChevronsDownUp strokeWidth={1.75} className="h-3.5 w-3.5" />
                )}
                {anyCollapsed ? 'Expand all' : 'Collapse all'}
              </button>
            )}
            {!fixedTaskType && (
              <div className="flex items-center gap-3 text-base">
                <button
                  onClick={() => setTaskType('daily')}
                  className={`transition-colors hover:opacity-80 ${
                    taskType === 'daily' ? 'font-medium' : 'opacity-60'
                  }`}
                  style={{
                    color: taskType === 'daily' ? 'rgb(247, 111, 83)' : '',
                  }}
                >
                  Daily
                </button>
                <span className="opacity-30">|</span>
                <button
                  onClick={() => setTaskType('longterm')}
                  className={`transition-colors hover:opacity-80 ${
                    taskType === 'longterm' ? 'font-medium' : 'opacity-60'
                  }`}
                  style={{
                    color: taskType === 'longterm' ? 'rgb(247, 111, 83)' : '',
                  }}
                >
                  Long-term
                </button>
              </div>
            )}
            <button
              onClick={() => setShowSettings(true)}
              className="ml-2 p-1.5 rounded-full hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
              title="Settings"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 20 20"
                fill="currentColor"
                className="w-4 h-4 text-gray-500"
              >
                <path
                  fillRule="evenodd"
                  d="M11.49 3.17c-.38-1.56-2.6-1.56-2.98 0a1.532 1.532 0 01-2.286.948c-1.372-.836-2.942.734-2.106 2.106.54.886.061 2.042-.947 2.287-1.561.379-1.561 2.6 0 2.978a1.532 1.532 0 01.947 2.287c-.836 1.372.734 2.942 2.106 2.106a1.532 1.532 0 012.287.947c.379 1.561 2.6 1.561 2.978 0a1.533 1.533 0 012.287-.947c1.372.836 2.942-.734 2.106-2.106a1.533 1.533 0 01.947-2.287c1.561-.379 1.561-2.6 0-2.978a1.532 1.532 0 01-.947-2.287c.836-1.372-.734-2.942-2.106-2.106a1.532 1.532 0 01-2.287-.947zM10 13a3 3 0 100-6 3 3 0 000 6z"
                  clipRule="evenodd"
                />
              </svg>
            </button>
          </div>
        )}
      </div>

      {error && (
        <div className="p-2 text-base text-red-600 bg-red-50 rounded-md">
          {error}
        </div>
      )}

      {/* Settings View */}
      {showSettings && !showArchived && (
        <div className="space-y-6 animate-slideIn">
          <div className="flex items-center justify-between p-4 bg-white/5 dark:bg-gray-800/20 rounded-lg">
            <div>
              <h3 className="text-base font-medium mb-1">Refresh daily tasks</h3>
              <p className="text-sm text-gray-500">
                Automatically refresh daily tasks at the start of each day.{' '}
              </p>
              <p className="text-sm text-gray-500">
                This helps you automatically re-setup daily tasks that you may
                want to work towards daily.
              </p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                className="sr-only peer"
                checked={dailyReset}
                onChange={handleToggleDailyReset}
                disabled={isTogglingReset}
              />
              <div
                className={`w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-orange-300 dark:peer-focus:ring-orange-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-orange-500 ${
                  isTogglingReset ? 'opacity-50 cursor-not-allowed' : ''
                }`}
              ></div>
            </label>
          </div>

          <button
            onClick={() => setShowArchived(true)}
            className="w-full p-4 text-left bg-white/5 dark:bg-gray-800/20 rounded-lg hover:bg-white/10 dark:hover:bg-gray-800/30 transition-colors"
          >
            <h3 className="text-base font-medium mb-1">View archived tasks</h3>
            <p className="text-sm text-gray-500">
              View and manage your archived tasks
            </p>
          </button>
        </div>
      )}

      {/* Archived Tasks View */}
      {showArchived && (
        <div className="space-y-4">
          {archivedTodos.length === 0 ? (
            <div className="py-4 text-center">
              <p className="text-gray-500 text-base">No archived tasks found.</p>
            </div>
          ) : (
            <ul className="space-y-4 divide-y divide-gray-200 dark:divide-gray-800/50">
              {archivedTodos.map((todo, index) => (
                <li
                  key={todo.id}
                  className="relative flex items-center justify-between group transition-all duration-200 opacity-60 pt-4 first:pt-0"
                >
                  <div className="flex items-center space-x-3 flex-1">
                    <div className="flex flex-col flex-1">
                      <label className="text-base cursor-default flex-1">
                        {todo.description}
                      </label>
                      {todo.scheduledTime && (
                        <div
                          className={`mt-1 text-sm flex items-center ${
                            isScheduledTimePast(todo.scheduledTime)
                              ? 'text-red-500'
                              : 'text-gray-500'
                          }`}
                        >
                          <svg
                            xmlns="http://www.w3.org/2000/svg"
                            viewBox="0 0 20 20"
                            fill="currentColor"
                            className={`w-3 h-3 mr-1 ${
                              isScheduledTimePast(todo.scheduledTime)
                                ? 'text-red-500'
                                : 'text-orange-400'
                            }`}
                          >
                            <path
                              fillRule="evenodd"
                              d="M10 18a8 8 0 100-16 8 8 0 000 16zm1-12a1 1 0 10-2 0v4a1 1 0 00.293.707l2.828 2.829a1 1 0 101.415-1.415L11 9.586V6z"
                              clipRule="evenodd"
                            />
                          </svg>
                          {formatScheduleTime(todo.scheduledTime)}
                        </div>
                      )}
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => handleMoveItem(todo.id, 'up')}
                      className="p-1 hover:bg-gray-100 dark:hover:bg-gray-800 rounded"
                      disabled={index === 0}
                    >
                      <ChevronUp className="h-4 w-4" />
                    </button>
                    <button
                      onClick={() => handleMoveItem(todo.id, 'down')}
                      className="p-1 hover:bg-gray-100 dark:hover:bg-gray-800 rounded"
                      disabled={index === archivedTodos.length - 1}
                    >
                      <ChevronDown className="h-4 w-4" />
                    </button>
                  </div>
                  <button
                    onClick={() => confirmDeleteTodo(todo)}
                    title="Delete task permanently"
                    className="opacity-0 group-hover:opacity-100 transition-opacity p-1.5 rounded-full hover:bg-gray-100 dark:hover:bg-gray-800"
                    style={{ color: 'rgb(247, 111, 83)' }}
                  >
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      viewBox="0 0 20 20"
                      fill="currentColor"
                      className="w-4 h-4"
                    >
                      <path
                        fillRule="evenodd"
                        d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z"
                        clipRule="evenodd"
                      />
                    </svg>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}

      {/* Delete Confirmation Modal */}
      {showDeleteModal && todoToDelete && (
        <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50">
          <div className="bg-white/5 backdrop-blur-md border border-white/10 rounded-lg p-6 max-w-md w-full mx-4 animate-slideIn">
            <h3 className="text-lg font-medium mb-2">Delete Task</h3>
            <p className="text-gray-300 mb-4">
              Are you sure you want to permanently delete "
              {todoToDelete.description}"? This action cannot be undone.
            </p>
            <div className="flex justify-end space-x-3">
              <button
                onClick={() => {
                  setShowDeleteModal(false);
                  setTodoToDelete(null);
                }}
                className="px-4 py-2 rounded bg-white/5 hover:bg-white/10 border border-white/10 text-base"
              >
                Cancel
              </button>
              <button
                onClick={handleDeleteConfirm}
                className="px-4 py-2 rounded bg-red-500/10 hover:bg-red-500/20 border border-red-500/20 text-red-500 text-base"
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Main Tasks View */}
      {!showSettings && !showArchived && (
        <div className="space-y-4">

          {orderedRows.length === 0 ? (
            <div className="py-4 text-center">
              <p className="text-gray-500 text-base">
                {enableTypeFilter && taskType === 'longterm' && listTypeFilter === 'note'
                  ? 'No notes yet.'
                  : enableTypeFilter && taskType === 'longterm' && listTypeFilter === 'task'
                  ? 'No tasks yet.'
                  : taskType === 'daily'
                  ? dailyAIOnly
                    ? 'No daily tasks yet. Use “Get Suggestion” below to generate some.'
                    : 'No daily tasks yet. Add one below!'
                  : taskType === 'longterm'
                  ? 'No items yet. Add one below!'
                  : 'No archived items found.'}
              </p>
            </div>
          ) : (
            // space-y-4 = 16px gap so the hover-menu has room above each row;
            // divide-y adds a subtle 1px line between rows for visual structure.
            // pl-8 reserves the chevron's gutter INSIDE the card: the button
            // hangs 32px left of each row, which would otherwise spill past
            // the card's own padding. The add form below carries the same
            // padding so its icon stays in the checkbox column.
            <ul className="space-y-4 divide-y divide-gray-200 dark:divide-gray-800/50 mt-4 pl-8">
              {renderedRows.map((todo, index) => (
                <li
                  key={todo.id}
                  tabIndex={0}
                  onKeyDown={(e) => handleRowKeyDown(e, todo.id)}
                  // pt-4 + first:pt-0 → content sits 16px below the divider
                  // line above it, mirroring the 16px gap below (space-y-4),
                  // so each row's content is centred between consecutive
                  // divider lines. First row has no line above → no top padding.
                  // focus-visible, not focus: rows are tabIndex=0 for Tab
                  // indent/outdent, so keyboard focus must show — but a mouse
                  // click on the row, its chevron or a hover action should not
                  // leave a ring drawn around the whole row.
                  className={`relative flex items-center justify-between group transition-all duration-200 outline-none focus-visible:ring-1 focus-visible:ring-primary/30 rounded pt-4 first:pt-0 ${
                    guideRail.get(todo.id) === 'child' ? 'ml-6' : ''
                  } ${
                    todo.parentId && revealedIds.current.has(todo.parentId)
                      ? 'animate-groupReveal'
                      : ''
                  } ${
                    newTodoAnimations[todo.id] ? 'animate-fadeIn' : ''
                  } ${taskType === 'archived' ? 'opacity-60' : ''}`}
                >
                  {/* Guide rail: a 1px hairline in the add bar's resting token,
                      on the checkbox column's centre (8px). The parent's piece
                      starts 4px beneath its checkbox, which is centred in the
                      content below pt-4 (no padding on the first row). Children
                      sit at ml-6 so their checkbox lines up with the parent's
                      text; each child's piece reaches up 17px through the
                      space-y-4 gap and its divider so the rail reads as one line. */}
                  {guideRail.has(todo.id) && !isCollapsed(todo.id) && (
                    <span
                      aria-hidden
                      data-guide-rail={guideRail.get(todo.id)}
                      className={cn(
                        'pointer-events-none absolute bottom-0 w-px bg-foreground/15',
                        guideRail.get(todo.id) === 'parent'
                          ? 'left-2 top-[calc(50%+20px)] group-first:top-[calc(50%+12px)]'
                          : '-left-4 -top-[17px]'
                      )}
                    />
                  )}
                  {/* Collapse chevron: in the gutter left of the checkbox
                      column so the text never moves, a 32px hit area centred
                      on the row content (below pt-4; none on the first row).
                      Keys stop here so the row's Tab handler can't swallow Tab
                      and trap focus, and a click never reaches the done toggle. */}
                  {childrenOf.has(todo.id) && (
                    <button
                      type="button"
                      aria-expanded={!isCollapsed(todo.id)}
                      aria-label={`${
                        isCollapsed(todo.id) ? 'Expand' : 'Collapse'
                      } ${todo.description}`}
                      onClick={(e) => {
                        e.stopPropagation();
                        toggleCollapsed(todo.id);
                      }}
                      onKeyDown={(e) => e.stopPropagation()}
                      className={cn(
                        'absolute -left-8 top-[calc(50%+8px)] group-first:top-1/2 flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-md text-foreground/40 outline-none transition-[color,background-color,opacity] duration-200 hover:text-primary focus-visible:bg-primary/10 focus-visible:text-primary',
                        // Hover devices: hidden until the row is hovered or
                        // focused. Touch: always there, quietly. Collapsed:
                        // always fully visible so the fold can be found.
                        isCollapsed(todo.id)
                          ? 'opacity-100'
                          : 'opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 [@media(hover:none)]:opacity-40'
                      )}
                    >
                      <ChevronRight
                        aria-hidden
                        className={cn(
                          'h-4 w-4 transition-transform duration-200',
                          !isCollapsed(todo.id) && 'rotate-90'
                        )}
                      />
                    </button>
                  )}
                  {editingId === todo.id ? (
                    <div className="flex items-center space-x-3 flex-1">
                      <div className="flex flex-1 space-x-2">
                        <input
                          ref={editInputRef}
                          type="text"
                          value={editText}
                          onChange={(e) => setEditText(e.target.value)}
                          onKeyDown={(e) => handleRowKeyDown(e, todo.id)}
                          className="flex-1 px-0 py-0 text-base bg-transparent border-b border-gray-300 dark:border-gray-600 focus:border-orange-500 dark:focus:border-orange-500 focus:outline-none"
                          style={{
                            color: todo.done ? 'rgb(247, 111, 83)' : '',
                            textDecoration: todo.done ? 'line-through' : 'none',
                            opacity: todo.done ? 0.7 : 1,
                          }}
                          disabled={isUpdating}
                        />
                        <button
                          onClick={() => updateTodoDescription(todo.id)}
                          className="px-2 py-1 text-sm rounded bg-white/5"
                          style={{ color: 'rgb(247, 111, 83)' }}
                          disabled={isUpdating}
                        >
                          {isUpdating ? 'Saving...' : 'Save'}
                        </button>
                        <button
                          onClick={cancelEditing}
                          className="px-2 py-1 text-sm rounded bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-800 dark:to-gray-900/80 border border-gray-200 dark:border-gray-700"
                          disabled={isUpdating}
                        >
                          Cancel
                        </button>
                      </div>
                    </div>
                  ) : schedulingId === todo.id ? (
                    <div className="flex items-center space-x-3 flex-1">
                      {(todo.type ?? 'task') === 'task' && (
                        <Checkbox
                          id={`todo-${todo.id}`}
                          checked={todo.done}
                          onCheckedChange={() => toggleTodo(todo.id)}
                        />
                      )}
                      <div className="flex flex-1 flex-wrap space-x-2">
                        <label
                          className={`text-base cursor-pointer flex-1 ${
                            todo.done ? 'line-through opacity-70' : ''
                          }`}
                        >
                          {todo.description}
                        </label>
                        <div className="mt-2 flex items-center space-x-2 w-full">
                          <DatePicker
                            selected={scheduleDate}
                            onChange={(date) => setScheduleDate(date)}
                            showTimeSelect
                            timeFormat="h:mm aa"
                            timeIntervals={15}
                            dateFormat="MMMM d, yyyy h:mm aa"
                            className="text-base p-2 border rounded flex-grow bg-transparent"
                            placeholderText="Select date and time"
                            disabled={isScheduling}
                          />
                          <button
                            onClick={scheduleTodo}
                            className="px-2 py-1 text-sm rounded bg-gradient-to-br from-orange-50 to-orange-100 dark:from-orange-900/40 dark:to-orange-800/60 border border-orange-200 dark:border-orange-800"
                            style={{ color: 'rgb(247, 111, 83)' }}
                            disabled={isScheduling || !scheduleDate}
                          >
                            {isScheduling ? 'Saving...' : 'Schedule'}
                          </button>
                          <button
                            onClick={cancelScheduling}
                            className="px-2 py-1 text-sm rounded bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-800 dark:to-gray-900/80 border border-gray-200 dark:border-gray-700"
                            disabled={isScheduling}
                          >
                            Cancel
                          </button>
                        </div>
                      </div>
                    </div>
                  ) : (
                    <>
                      <div
                        className="flex items-center space-x-3 flex-1"
                        onClick={
                          (todo.type ?? 'task') === 'task'
                            ? () => toggleTodo(todo.id)
                            : undefined
                        }
                        style={{
                          cursor:
                            (todo.type ?? 'task') === 'task' ? 'pointer' : 'default',
                        }}
                      >
                        <div className="flex flex-1 items-center space-x-2">
                          {(todo.type ?? 'task') === 'task' && (
                            <input
                              type="checkbox"
                              checked={todo.done}
                              onChange={() => toggleTodo(todo.id)}
                              className="w-4 h-4 rounded border-gray-300 dark:border-gray-600"
                              style={{
                                color: 'rgb(247, 111, 83)',
                                accentColor: 'rgb(247, 111, 83)',
                              }}
                              disabled={isUpdating}
                            />
                          )}
                          <div className="flex flex-col flex-1">
                            {/* The count sits right after the text, so the
                                label no longer stretches (flex-1). */}
                            <div className="flex items-baseline gap-2">
                              <label
                                className={`text-base cursor-pointer ${
                                  todo.done ? 'line-through opacity-70' : ''
                                } ${
                                  todo.type === 'note'
                                    ? 'italic text-gray-400 dark:text-gray-500'
                                    : ''
                                } ${
                                  newTodoAnimations[todo.id] ? 'relative' : ''
                                }`}
                              >
                                {todo.description}
                              </label>
                              {isCollapsed(todo.id) && (
                                <span className="shrink-0 text-sm tabular-nums text-foreground/40 animate-in fade-in duration-200">
                                  {collapsedCount(childrenOf.get(todo.id)!)}
                                </span>
                              )}
                            </div>
                            {todo.scheduledTime && (
                              <div
                                className={`mt-1 text-sm flex items-center ${
                                  isScheduledTimePast(todo.scheduledTime)
                                    ? 'text-red-500'
                                    : 'text-gray-500'
                                }`}
                              >
                                <svg
                                  xmlns="http://www.w3.org/2000/svg"
                                  viewBox="0 0 20 20"
                                  fill="currentColor"
                                  className={`w-3 h-3 mr-1 ${
                                    isScheduledTimePast(todo.scheduledTime)
                                      ? 'text-red-500'
                                      : 'text-orange-400'
                                  }`}
                                >
                                  <path
                                    fillRule="evenodd"
                                    d="M10 18a8 8 0 100-16 8 8 0 000 16zm1-12a1 1 0 10-2 0v4a1 1 0 00.293.707l2.828 2.829a1 1 0 101.415-1.415L11 9.586V6z"
                                    clipRule="evenodd"
                                  />
                                </svg>
                                {formatScheduleTime(todo.scheduledTime)}
                              </div>
                            )}
                          </div>
                        </div>
                      </div>

                      {/* Hover menu floats in the gap ABOVE the row. -top-4
                          (-16px) lines its top edge up with the bottom of the
                          previous row's content, so it sits in the 16px gap
                          plus 12px into this row's pt-4 empty area. No
                          overlap with previous content. */}
                      <div className="absolute -top-4 right-0 flex space-x-1 z-10">
                        {taskType === 'archived' ? (
                          // Delete button for archived items
                          <button
                            onClick={() => deleteTodo(todo.id)}
                            title="Delete task permanently"
                            className="opacity-0 group-hover:opacity-100 transition-opacity p-1.5 rounded-full hover:bg-gray-100 dark:hover:bg-gray-800"
                            style={{ color: 'rgb(247, 111, 83)' }}
                          >
                            <svg
                              xmlns="http://www.w3.org/2000/svg"
                              viewBox="0 0 20 20"
                              fill="currentColor"
                              className="w-4 h-4"
                            >
                              <path
                                fillRule="evenodd"
                                d="M8.75 1A2.75 2.75 0 006 3.75v.443c-.795.077-1.584.176-2.365.298a.75.75 0 10.23 1.482l.149-.022.841 10.518A2.75 2.75 0 007.596 19h4.807a2.75 2.75 0 002.742-2.53l.841-10.52.149.023a.75.75 0 00.23-1.482A41.03 41.03 0 0014 4.193V3.75A2.75 2.75 0 0011.25 1h-2.5zM10 4c.84 0 1.673.025 2.5.075V3.75c0-.69-.56-1.25-1.25-1.25h-2.5c-.69 0-1.25.56-1.25 1.25v.325C8.327 4.025 9.16 4 10 4zM8.58 7.72a.75.75 0 00-1.5.06l.3 7.5a.75.75 0 101.5-.06l-.3-7.5zm6.34.06a.75.75 0 10-1.5-.06l-.3 7.5a.75.75 0 101.5.06l.3-7.5z"
                                clipRule="evenodd"
                              />
                            </svg>
                          </button>
                        ) : (
                          <>
                            {/* Type toggle (task ↔ note) */}
                            <button
                              onClick={() => toggleTodoType(todo.id)}
                              title={
                                (todo.type ?? 'task') === 'note'
                                  ? 'Convert to task'
                                  : 'Convert to note'
                              }
                              className="opacity-0 group-hover:opacity-100 transition-opacity p-1.5 rounded-full hover:bg-gray-100 dark:hover:bg-gray-800"
                              style={{ color: 'rgb(247, 111, 83)' }}
                            >
                              {(todo.type ?? 'task') === 'note' ? (
                                <CheckSquare className="w-4 h-4" />
                              ) : (
                                <FileText className="w-4 h-4" />
                              )}
                            </button>

                            {/* Indent / outdent — Tab equivalent.
                                If row is already nested, the button outdents
                                (Shift+Tab equivalent). */}
                            <button
                              onClick={() =>
                                todo.parentId
                                  ? outdentTodo(todo.id)
                                  : indentTodo(todo.id)
                              }
                              title={
                                todo.parentId
                                  ? 'Outdent to top level (Shift+Tab)'
                                  : 'Indent under previous row (Tab)'
                              }
                              className="opacity-0 group-hover:opacity-100 transition-opacity p-1.5 rounded-full hover:bg-gray-100 dark:hover:bg-gray-800"
                              style={{ color: 'rgb(247, 111, 83)' }}
                            >
                              <ChevronRight
                                className={`w-4 h-4 transition-transform ${
                                  todo.parentId ? 'rotate-180' : ''
                                }`}
                              />
                            </button>

                            {/* Schedule Icon */}
                            <button
                              onClick={() => startScheduling(todo)}
                              title="Schedule this task"
                              className="opacity-0 group-hover:opacity-100 transition-opacity p-1.5 rounded-full hover:bg-gray-100 dark:hover:bg-gray-800"
                              style={{
                                color: todo.scheduledTime
                                  ? 'rgb(247, 111, 83)'
                                  : 'rgb(150, 150, 150)',
                              }}
                            >
                              <svg
                                xmlns="http://www.w3.org/2000/svg"
                                viewBox="0 0 20 20"
                                fill="currentColor"
                                className="w-4 h-4"
                              >
                                <path
                                  fillRule="evenodd"
                                  d="M10 18a8 8 0 100-16 8 8 0 000 16zm1-12a1 1 0 10-2 0v4a1 1 0 00.293.707l2.828 2.829a1 1 0 101.415-1.415L11 9.586V6z"
                                  clipRule="evenodd"
                                />
                              </svg>
                            </button>

                            {/* Edit Icon */}
                            <button
                              onClick={() => startEditing(todo)}
                              title="Edit task"
                              className="opacity-0 group-hover:opacity-100 transition-opacity p-1.5 rounded-full hover:bg-gray-100 dark:hover:bg-gray-800"
                              style={{ color: 'rgb(247, 111, 83)' }}
                            >
                              <svg
                                xmlns="http://www.w3.org/2000/svg"
                                viewBox="0 0 20 20"
                                fill="currentColor"
                                className="w-4 h-4"
                              >
                                <path d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z" />
                              </svg>
                            </button>

                            {/* Archive Icon */}
                            <button
                              onClick={() => archiveTodo(todo.id)}
                              title="Archive task"
                              className="opacity-0 group-hover:opacity-100 transition-opacity p-1.5 rounded-full hover:bg-gray-100 dark:hover:bg-gray-800"
                              style={{ color: 'rgb(247, 111, 83)' }}
                            >
                              <svg
                                xmlns="http://www.w3.org/2000/svg"
                                viewBox="0 0 20 20"
                                fill="currentColor"
                                className="w-4 h-4"
                              >
                                <path d="M2 3a1 1 0 00-1 1v1a1 1 0 001 1h16a1 1 0 001-1V4a1 1 0 00-1-1H2z" />
                                <path d="M2 7a1 1 0 00-1 1v10a1 1 0 001 1h16a1 1 0 001-1V8a1 1 0 00-1-1H2zm0 2h16v8H2V9z" />
                              </svg>
                            </button>
                          </>
                        )}
                      </div>
                    </>
                  )}
                </li>
              ))}
            </ul>
          )}

          {/* Page controls, bottom-right below the list and above the add
              bar's row of air (R5). Quiet by default, warming to primary on
              hover and focus like the header's collapse-all toggle. Nothing at
              all while everything fits on one page (R6). pr-1 keeps the last
              chevron off the card's edge; pl-8 matches the list's gutter so
              the row starts in the same column. */}
          {totalPages > 1 && (
            <div className="flex items-center justify-end gap-2 pl-8 pr-1 text-sm text-foreground/50">
              <button
                type="button"
                onClick={() => setPage(currentPage - 1)}
                disabled={currentPage === 1}
                aria-label="Previous page"
                className={pageButtonClass}
              >
                <ChevronLeft strokeWidth={1.75} className="h-4 w-4" />
              </button>
              {/* The visible count is decoration; the live region beside it
                  carries the announcement so it reads as a sentence (R12). */}
              <span data-testid="page-counter" aria-hidden="true">
                {currentPage} of {totalPages}
              </span>
              <span role="status" aria-live="polite" className="sr-only">
                Page {currentPage} of {totalPages}
              </span>
              <button
                type="button"
                onClick={() => setPage(currentPage + 1)}
                disabled={currentPage === totalPages}
                aria-label="Next page"
                className={pageButtonClass}
              >
                <ChevronRight strokeWidth={1.75} className="h-4 w-4" />
              </button>
            </div>
          )}

          {/* Add form — moved to the bottom so adding a row reads as
              "append to list", and Tab on the new row indents under the
              one above. Hidden in archived view and on the daily side
              when dailyAIOnly is on (only AI suggestions populate dailies). */}
          {taskType !== 'archived' && !(dailyAIOnly && taskType === 'daily') && (
            <div className="space-y-3 pt-3 pb-6 pl-8">
              {/* One hairline runs under the whole row (icon, text, button) so
                  everything sits on the same line with py-3 of air above it.
                  On focus an ember underline draws in from the left over it. */}
              <form
                onSubmit={addTodo}
                className={cn(
                  'group/add relative flex items-center border-b border-foreground/15 py-3 transition-[margin,border-color] duration-300 focus-within:border-transparent',
                  addBarParent && 'ml-6'
                )}
              >
                {/* Nested under the last group, the bar joins its guide rail:
                    at ml-6, -left-4 is the checkbox column (8px). The piece
                    reaches 28px up (space-y-4 + pt-3) to the list's bottom
                    edge and 28px down (py-3 + half of h-8) to the icon's centre. */}
                {addBarRailParentId && (
                  <span
                    aria-hidden
                    data-guide-rail="add-bar"
                    className="pointer-events-none absolute -left-4 -top-7 h-14 w-px bg-foreground/15 animate-in fade-in duration-300"
                  />
                )}
                <span
                  aria-hidden
                  className="pointer-events-none absolute inset-x-0 -bottom-px h-px origin-left scale-x-0 bg-gradient-to-r from-primary via-primary/60 to-primary/0 shadow-[0_0_10px_rgba(247,111,83,0.45)] transition-transform duration-500 [transition-timing-function:cubic-bezier(0.22,1,0.36,1)] group-focus-within/add:scale-x-100"
                />

                {/* Type toggle for the next item to be created. -ml-2 on a w-8
                    hit area centres the glyph over the list's checkbox column,
                    so the typed text starts where row text starts (24px). */}
                <button
                  type="button"
                  onClick={() =>
                    setNewTodoType((t) => (t === 'task' ? 'note' : 'task'))
                  }
                  title={
                    newTodoType === 'task'
                      ? 'Creating a task — click to switch to note'
                      : 'Creating a note — click to switch to task'
                  }
                  aria-label={
                    newTodoType === 'task'
                      ? 'Switch to adding a note'
                      : 'Switch to adding a task'
                  }
                  className="-ml-2 grid h-8 w-8 shrink-0 place-items-center rounded-full text-foreground/40 transition-colors duration-300 hover:bg-primary/10 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 group-focus-within/add:text-primary"
                >
                  {/* Both glyphs share one grid cell and cross-fade on swap. */}
                  <CheckSquare
                    strokeWidth={1.75}
                    className={cn(
                      'col-start-1 row-start-1 h-[18px] w-[18px] transition-all duration-300',
                      newTodoType === 'task'
                        ? 'rotate-0 scale-100 opacity-100'
                        : '-rotate-90 scale-75 opacity-0'
                    )}
                  />
                  <FileText
                    strokeWidth={1.75}
                    className={cn(
                      'col-start-1 row-start-1 h-[18px] w-[18px] transition-all duration-300',
                      newTodoType === 'note'
                        ? 'rotate-0 scale-100 opacity-100'
                        : 'rotate-90 scale-75 opacity-0'
                    )}
                  />
                </button>
                {/* readOnly, not disabled, while saving: a disabled input
                    drops focus, which would break rapid back-to-back entry.
                    addTodo already ignores submits while one is in flight. */}
                <input
                  ref={newTodoInputRef}
                  type="text"
                  value={newTodo}
                  onChange={(e) => setNewTodo(e.target.value)}
                  onFocus={maybeShowTabHint}
                  onKeyDown={handleAddBarKeyDown}
                  placeholder={
                    addBarParent
                      ? `${newTodoType === 'note' ? 'Jot a note' : 'Add'} under “${addBarParentLabel}”…`
                      : newTodoType === 'note'
                      ? 'Jot down a note…'
                      : 'Add something to do…'
                  }
                  aria-label={`${newTodoType === 'note' ? 'New note' : 'New task'}${
                    addBarParent ? ` under “${addBarParent.description}”` : ''
                  }`}
                  className="h-8 min-w-0 flex-1 bg-transparent text-base leading-8 text-foreground caret-primary placeholder:italic placeholder:text-foreground/35 focus:outline-none read-only:opacity-60"
                  readOnly={isSubmitting || isTyping}
                />
                {/* Quiet until there's something to add, then it warms up. */}
                <button
                  type="submit"
                  className={cn(
                    'ml-3 inline-flex h-8 shrink-0 items-center gap-1.5 rounded-full px-3.5 text-sm font-medium transition-all duration-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 disabled:pointer-events-none disabled:opacity-60',
                    newTodo.trim()
                      ? 'bg-primary text-primary-foreground shadow-[0_2px_14px_-3px_rgba(247,111,83,0.6)] hover:bg-primary/90'
                      : 'text-foreground/40 hover:bg-primary/10 hover:text-primary'
                  )}
                  disabled={isSubmitting || isTyping}
                >
                  <Plus strokeWidth={2.25} className="h-3.5 w-3.5" />
                  {isSubmitting ? 'Adding…' : 'Add'}
                </button>
              </form>

              {/* The "Get Suggestion" trigger and its result panel are gone for
                  now — the space mattered more than the button. getAISuggestion
                  / useSuggestion and their state above are kept, unwired, for
                  whatever surface this capability gets next. pb-6 on the wrapper
                  keeps air under the add bar where the button used to sit. */}
            </div>
          )}

          {/* Daily Insights Section - Only show for daily view */}
          {taskType === 'daily' && (todos?.length > 0 || dailyAIOnly) && (
            <div className="mt-8 border-t border-gray-100 dark:border-gray-800 pt-4">
              <div className="flex justify-between items-center mb-2">
                <div className="flex items-center gap-2">
                  <h3 className="text-base font-medium">Suggested Daily Tasks</h3>
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
                    <div className="absolute left-1/2 -translate-x-1/2 bottom-full mb-2 w-64 opacity-0 group-hover:opacity-100 transition-opacity duration-200 pointer-events-none">
                      <div className="bg-white/5 backdrop-blur-sm border border-white/10 rounded-lg p-3 shadow-lg">
                        <p className="text-base text-white mb-1">
                          Focus on what matters most to you
                        </p>
                        <p className="text-sm text-gray-400">
                          Daily suggestions are driven by your long-term goals
                          and priorities
                        </p>
                      </div>
                      <div className="absolute left-1/2 -translate-x-1/2 -bottom-1 w-2 h-2 bg-white/5 border-r border-b border-white/10 transform rotate-45"></div>
                    </div>
                  </div>
                </div>
                <button
                  onClick={toggleInsightsVisibility}
                  className="text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
                >
                  {showInsights ? 'Hide suggestions' : 'Show suggestions'}
                </button>
              </div>

              {showInsights && (
                <div className="space-y-3 p-3 bg-white/5 dark:bg-gray-800/20 rounded-md">
                  <div className="flex justify-end">
                    <span className="text-sm text-gray-500">
                      Based on long-term goals
                    </span>
                  </div>

                  <ul className="space-y-2">
                    {dailyInsights.map((insight, index) => (
                      <li
                        key={`insight-${index}`}
                        className={`flex items-center justify-between transition-all duration-500 ${
                          visibleInsights[index]
                            ? 'opacity-100 translate-y-0'
                            : 'opacity-0 translate-y-2'
                        }`}
                      >
                        <span className="text-base text-gray-300">{insight}</span>
                        <button
                          onClick={() => addInsightAsTodo(insight)}
                          className="text-sm px-2 py-1 rounded bg-white/5 hover:bg-white/10 dark:hover:bg-gray-700/50"
                          style={{ color: 'rgb(247, 111, 83)' }}
                        >
                          Add
                        </button>
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          )}
        </div>
      )}

      <style jsx global>{`
        /* DatePicker Dark Theme Styles */
        .react-datepicker {
          background-color: rgb(17, 24, 39) !important;
          border: 1px solid rgba(255, 255, 255, 0.1) !important;
          border-radius: 0.5rem !important;
          font-family: inherit !important;
          position: relative !important;
        }

        .react-datepicker::before {
          content: '' !important;
          position: absolute !important;
          inset: 0 !important;
          background-color: rgba(255, 255, 255, 0.05) !important;
          border-radius: 0.5rem !important;
          pointer-events: none !important;
        }

        .react-datepicker__header {
          background-color: rgb(17, 24, 39) !important;
          border-bottom: 1px solid rgba(255, 255, 255, 0.1) !important;
          padding-top: 0.5rem !important;
          position: relative !important;
        }

        .react-datepicker__header::before {
          content: '' !important;
          position: absolute !important;
          inset: 0 !important;
          background-color: rgba(255, 255, 255, 0.05) !important;
          pointer-events: none !important;
        }

        .react-datepicker__time-box {
          background-color: rgb(17, 24, 39) !important;
          position: relative !important;
        }

        .react-datepicker__time-box::before {
          content: '' !important;
          position: absolute !important;
          inset: 0 !important;
          background-color: rgba(255, 255, 255, 0.05) !important;
          pointer-events: none !important;
        }

        .react-datepicker__current-month,
        .react-datepicker__day-name,
        .react-datepicker__day {
          color: rgb(229, 231, 235) !important;
        }

        .react-datepicker__day:hover {
          background-color: rgba(255, 255, 255, 0.1) !important;
        }

        .react-datepicker__day--selected {
          background-color: rgb(247, 111, 83) !important;
          color: white !important;
        }

        .react-datepicker__day--keyboard-selected {
          background-color: rgba(247, 111, 83, 0.5) !important;
        }

        .react-datepicker__time-container {
          border-left: 1px solid rgba(255, 255, 255, 0.1) !important;
        }

        .react-datepicker__time-list-item {
          color: rgb(229, 231, 235) !important;
        }

        .react-datepicker__time-list-item:hover {
          background-color: rgba(255, 255, 255, 0.1) !important;
        }

        .react-datepicker__time-list-item--selected {
          background-color: rgb(247, 111, 83) !important;
          color: white !important;
        }

        .react-datepicker__navigation {
          top: 0.5rem !important;
        }

        .react-datepicker__navigation-icon::before {
          border-color: rgb(229, 231, 235) !important;
        }

        .react-datepicker__navigation:hover *::before {
          border-color: rgb(247, 111, 83) !important;
        }

        .react-datepicker__day--disabled {
          color: rgba(229, 231, 235, 0.3) !important;
        }

        .react-datepicker__day--outside-month {
          color: rgba(229, 231, 235, 0.5) !important;
        }

        .react-datepicker__input-container input {
          background-color: transparent !important;
          color: rgb(229, 231, 235) !important;
          border: 1px solid rgba(255, 255, 255, 0.1) !important;
          border-radius: 0.375rem !important;
          padding: 0.5rem !important;
          width: 100% !important;
        }

        .react-datepicker__input-container input:focus {
          outline: none !important;
          border-color: rgb(247, 111, 83) !important;
        }

        /* Animation for the popup */
        .react-datepicker-popper {
          z-index: 50 !important;
          animation: datepickerFadeIn 0.2s ease-out !important;
        }

        @keyframes datepickerFadeIn {
          from {
            opacity: 0;
            transform: translateY(-10px);
          }
          to {
            opacity: 1;
            transform: translateY(0);
          }
        }
      `}</style>
    </div>
  );
}
