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
  getChecklistSuggestion,
  scope,
  ScopeEnum,
  getDailyInsights,
  archiveChecklistItem,
  fetchArchivedChecklist,
  ChecklistResponse,
  ChecklistSuggestionResponse,
  toggleDailyReset,
  fetchPlanDetails,
} from '@/services/api';
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
  ChevronRight,
  ArrowUpDown,
  Info,
  CheckSquare,
  FileText,
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
        'Press Tab after typing to nest the new item under the one above. Shift+Tab to outdent.',
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

  // Type to use when creating the next item via the add form.
  const [newTodoType, setNewTodoType] = useState<'task' | 'note'>('task');

  const [newTodo, setNewTodo] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

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
        let response: ChecklistResponse;

        if (taskType === 'archived') {
          response = await fetchArchivedChecklist(planId, 'daily');
        } else {
          response = await fetchChecklist(
            planId,
            taskType as 'daily' | 'longterm'
          );
        }

        if (cancelled) return;
        setTodos(response.result || []);
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
      if (response.result !== 'success') {
        // Revert if failed
        setTodos(
          todos?.map((todo) =>
            todo.id === id ? { ...todo, done: todoToToggle.done } : todo
          )
        );
        setError('Failed to update task status. Please try again.');
      }
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

      if (response.result !== 'success') {
        // Revert if failed
        setTodos(
          todos?.map((todo) =>
            todo.id === schedulingId
              ? { ...todo, scheduledTime: originalScheduledTime }
              : todo
          )
        );
        setError('Failed to schedule task. Please try again.');
      }
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

      if (response.result !== 'success') {
        // Revert if failed
        setTodos(
          todos?.map((todo) =>
            todo.id === id ? { ...todo, description: originalText } : todo
          )
        );
        setError('Failed to update task. Please try again.');
      }
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
        { type: newTodoType }
      );
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
    }
  };

  // Get AI suggestion for a new todo
  const getAISuggestion = async () => {
    if (isFetchingSuggestion || isTyping || isSuggestionTyping) return;

    try {
      setIsFetchingSuggestion(true);
      const response: ChecklistSuggestionResponse =
        await getChecklistSuggestion(planId, taskType as 'daily' | 'longterm');
      setSuggestion(response.result);
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
      const longTermResponse = await fetchChecklist(planId, 'longterm');
      if (!longTermResponse.result || longTermResponse.result.length === 0) {
        // No long-term items, so no need to fetch insights
        setDailyInsights([]);
        setVisibleInsights([]);
        return;
      }

      const response = await getDailyInsights(planId);
      if (response && response.result) {
        setDailyInsights(response.result);
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
      if (response.result !== 'success') {
        // Restore if archiving failed
        setTodos((prevTodos) => [...prevTodos, todoToArchive]);
        setArchivedTodos(archivedTodos?.filter((todo) => todo.id !== id));
        setError('Failed to archive task. Please try again.');
      }
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
      if (response.result !== 'success') {
        setTodos((prev) =>
          prev.map((t) =>
            t.id === id ? { ...t, type: previous.type, done: previous.done } : t
          )
        );
        setError('Failed to change item type. Please try again.');
      }
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

  // Render order: top-level rows in their existing order, each followed by
  // its children (also in existing order). Children whose parent isn't in
  // the filtered set fall through as top-level (no visual indent).
  const orderedRows = useMemo(() => {
    if (!filteredTodos) return [];
    const visibleIds = new Set(filteredTodos.map((t) => t.id));
    const tops = filteredTodos.filter((t) => !t.parentId || !visibleIds.has(t.parentId));
    const result: ChecklistItem[] = [];
    for (const top of tops) {
      result.push(top);
      for (const t of filteredTodos) {
        if (t.parentId === top.id) result.push(t);
      }
    }
    return result;
  }, [filteredTodos]);

  // Track which parent IDs are actually rendered so children of out-of-view
  // parents drop their visual indent.
  const renderedParents = useMemo(
    () => new Set(orderedRows.filter((r) => !r.parentId).map((r) => r.id)),
    [orderedRows]
  );

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
      if (response.result !== 'success') {
        setTodos((prev) =>
          prev.map((t) =>
            t.id === id ? { ...t, parentId: previousParentId } : t
          )
        );
        setError('Failed to indent item. Please try again.');
      }
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
      if (response.result !== 'success') {
        setTodos((prev) =>
          prev.map((t) =>
            t.id === id ? { ...t, parentId: previousParentId } : t
          )
        );
        setError('Failed to outdent item. Please try again.');
      }
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
      const response = await fetchArchivedChecklist(planId, 'daily');
      setArchivedTodos(response.result || []);
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

  // Fetch plan details including dailyReset
  useEffect(() => {
    const loadPlanDetails = async () => {
      if (!planId) return;

      console.log('@DailyReset fetching plan details');
      try {
        const [checklistResponse, planResponse] = await Promise.all([
          fetchChecklist(planId, 'daily'),
          fetchPlanDetails(planId),
        ]);
        setTodos(checklistResponse.result);
        console.log('@DailyReset daily reset with:', planResponse.result);
        setDailyReset(planResponse.result.dailyReset);
      } catch (error) {
        console.log('@DailyReset Error fetching plan details');
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
      const response = await toggleDailyReset(planId);
      // Only revert if we get a non-200 status code
      if (response.statusCode !== 200) {
        // Revert on failure
        setDailyReset((prev) => !prev);
        throw new Error('Failed to toggle daily reset');
      }
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
            <ul className="space-y-4 divide-y divide-gray-200 dark:divide-gray-800/50 mt-4">
              {orderedRows.map((todo, index) => (
                <li
                  key={todo.id}
                  tabIndex={0}
                  onKeyDown={(e) => handleRowKeyDown(e, todo.id)}
                  // pt-4 + first:pt-0 → content sits 16px below the divider
                  // line above it, mirroring the 16px gap below (space-y-4),
                  // so each row's content is centred between consecutive
                  // divider lines. First row has no line above → no top padding.
                  className={`relative flex items-center justify-between group transition-all duration-200 outline-none focus:ring-1 focus:ring-orange-500/30 rounded pt-4 first:pt-0 ${
                    todo.parentId && renderedParents.has(todo.parentId)
                      ? 'ml-8 border-l-2 border-white/10 pl-3'
                      : ''
                  } ${
                    newTodoAnimations[todo.id] ? 'animate-fadeIn' : ''
                  } ${taskType === 'archived' ? 'opacity-60' : ''}`}
                >
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
                            <label
                              className={`text-base cursor-pointer flex-1 ${
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

          {/* Add form — moved to the bottom so adding a row reads as
              "append to list", and Tab on the new row indents under the
              one above. Hidden in archived view and on the daily side
              when dailyAIOnly is on (only AI suggestions populate dailies). */}
          {taskType !== 'archived' && !(dailyAIOnly && taskType === 'daily') && (
            <div className="space-y-3 pt-2">
              <form onSubmit={addTodo} className="flex items-center space-x-2">
                {/* Type toggle for the next item to be created */}
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
                  className="p-1.5 rounded-full hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
                  style={{ color: 'rgb(247, 111, 83)' }}
                >
                  {newTodoType === 'task' ? (
                    <CheckSquare className="w-4 h-4" />
                  ) : (
                    <FileText className="w-4 h-4" />
                  )}
                </button>
                <input
                  type="text"
                  value={newTodo}
                  onChange={(e) => setNewTodo(e.target.value)}
                  onFocus={maybeShowTabHint}
                  placeholder={
                    newTodoType === 'note' ? 'Add a note...' : 'Add a new task...'
                  }
                  className="flex-1 px-0 py-0 text-base bg-transparent border-b border-gray-300 dark:border-gray-600 focus:border-orange-500 dark:focus:border-orange-500 focus:outline-none"
                  disabled={isSubmitting || isTyping}
                />
                <button
                  type="submit"
                  className="inline-flex items-center gap-1.5 px-5 py-2.5 rounded-md bg-primary text-white font-semibold shadow-sm transition-colors hover:bg-primary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 disabled:opacity-50 disabled:pointer-events-none"
                  disabled={isSubmitting || isTyping}
                >
                  <Plus className="w-4 h-4" />
                  {isSubmitting ? 'Adding...' : 'Add'}
                </button>
              </form>

              <div className="flex justify-between items-center">
                <button
                  onClick={getAISuggestion}
                  disabled={
                    isFetchingSuggestion || isTyping || isSuggestionTyping
                  }
                  className="text-base text-gray-600 flex items-center px-3 py-1.5 rounded-md transition-colors bg-white/5"
                >
                  {isFetchingSuggestion ? (
                    'Getting suggestion...'
                  ) : (
                    <>
                      <svg
                        xmlns="http://www.w3.org/2000/svg"
                        viewBox="0 0 20 20"
                        fill="currentColor"
                        className="w-4 h-4 mr-1.5"
                      >
                        <path d="M10 2a8 8 0 1 0 0 16 8 8 0 0 0 0-16zm-.707 9.293a1 1 0 0 1 0 1.414 1 1 0 0 1-1.414 0l-2-2a1 1 0 0 1 1.414-1.414L9 10.586l3.293-3.293a1 1 0 0 1 1.414 1.414l-4 4z" />
                      </svg>
                      Get Suggestion
                    </>
                  )}
                </button>
              </div>

              {/* AI Suggestion Component */}
              {suggestion && (
                <div className="mt-2 p-3 text-base flex items-center px-3 py-1.5 rounded-md transition-colors bg-white/5 dark:bg-gray-900/10">
                  <div className="flex justify-between items-center w-full">
                    <div className="flex items-start">
                      <div className="ml-2 text-gray-600 text-base">
                        <p className="text-gray-600 font-medium">
                          generated suggestion
                        </p>
                        <p className="mt-1">
                          {isSuggestionTyping ? displaySuggestion : suggestion}
                          {isSuggestionTyping && (
                            <span className="animate-pulse">|</span>
                          )}
                        </p>
                      </div>
                    </div>
                    <button
                      onClick={useSuggestion}
                      disabled={isTyping || isSuggestionTyping}
                      className="ml-4 px-2.5 py-0.5 h-[30px] text-sm font-medium rounded bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-800 dark:to-gray-900/80 border border-gray-200 dark:border-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors flex-shrink-0"
                    >
                      Use suggestion
                    </button>
                  </div>
                </div>
              )}
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
