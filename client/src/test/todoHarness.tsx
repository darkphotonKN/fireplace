import { beforeEach, vi } from 'vitest';
import { fireEvent, render, screen, within } from '@testing-library/react';
import type { ComponentProps } from 'react';
import type Todo from '@/components/Todo';

/**
 * The shared harness for the `Todo.*.test.tsx` files.
 *
 * Importing this module mocks everything `<Todo>` talks to (routing, toasts,
 * the API adapters) and resets those mocks before each test. Each mock is a
 * `vi.fn()` exported here, so a test configures or inspects it directly:
 *
 *   import { api, item, renderTodo } from '@/test/todoHarness';
 *   api.updateChecklistItem.mockRejectedValue(new Error('nope'));
 *
 * Defaults resolve to something harmless; a test that cares sets its own.
 */

/** A mocked adapter call: any arguments, a promise of whatever the test needs. */
type ApiFn = (...args: unknown[]) => Promise<unknown>;
// A fresh copy per call, so nothing a test or Todo does to one result leaks into the next.
const apiFn = (fallback: unknown = undefined) =>
  vi.fn<ApiFn>(async () => structuredClone(fallback));

export const api = {
  fetchChecklist: apiFn([]),
  fetchArchivedChecklist: apiFn([]),
  createChecklistItem: apiFn(),
  updateChecklistItem: apiFn({}),
  deleteChecklistItem: apiFn(),
  updateChecklistDates: apiFn({}),
  archiveChecklistItem: apiFn({}),
  reorderChecklistItems: apiFn([]),
  scheduleChecklistItem: apiFn({}),
  duplicateChecklistItem: apiFn([]),
  getChecklistSuggestion: apiFn(''),
  getPlan: apiFn({ id: 'plan-1', dailyReset: false }),
  listPlanMembers: apiFn([]),
  addPlanMember: apiFn(),
  removePlanMember: apiFn(),
};

export const toast = vi.fn();
export const router = { push: vi.fn(), replace: vi.fn(), refresh: vi.fn() };

vi.mock('next/navigation', () => ({
  useParams: () => ({ planId: 'plan-1' }),
  useRouter: () => router,
}));

vi.mock('@/components/ui/use-toast', () => ({
  useToast: () => ({ toast }),
}));

vi.mock('@/services/api', () => ({
  fetchChecklist: (...a: unknown[]) => api.fetchChecklist(...a),
  fetchArchivedChecklist: (...a: unknown[]) => api.fetchArchivedChecklist(...a),
  createChecklistItem: (...a: unknown[]) => api.createChecklistItem(...a),
  updateChecklistItem: (...a: unknown[]) => api.updateChecklistItem(...a),
  deleteChecklistItem: (...a: unknown[]) => api.deleteChecklistItem(...a),
  updateChecklistDates: (...a: unknown[]) => api.updateChecklistDates(...a),
  archiveChecklistItem: (...a: unknown[]) => api.archiveChecklistItem(...a),
  reorderChecklistItems: (...a: unknown[]) => api.reorderChecklistItems(...a),
  scheduleChecklistItem: (...a: unknown[]) => api.scheduleChecklistItem(...a),
  scope: { DAILY: 'daily', LONGTERM: 'longterm' },
  ScopeEnum: { DAILY: 'daily', LONGTERM: 'longterm' },
}));

vi.mock('@/api/checklists', async (importOriginal) => ({
  ...(await importOriginal<object>()),
  duplicateChecklistItem: (...a: unknown[]) => api.duplicateChecklistItem(...a),
}));

vi.mock('@/api/insights', () => ({
  getChecklistSuggestion: (...a: unknown[]) => api.getChecklistSuggestion(...a),
}));

vi.mock('@/api/plans', () => ({
  getPlan: (...a: unknown[]) => api.getPlan(...a),
  listPlanMembers: (...a: unknown[]) => api.listPlanMembers(...a),
  addPlanMember: (...a: unknown[]) => api.addPlanMember(...a),
  removePlanMember: (...a: unknown[]) => api.removePlanMember(...a),
}));

let nextId = 0;

beforeEach(() => {
  // mockReset puts each fn back on the default it was created with.
  Object.values(api).forEach((fn) => fn.mockReset());
  toast.mockReset();
  Object.values(router).forEach((fn) => fn.mockReset());
  nextId = 0;
  window.localStorage.clear();
});

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

export type TestItem = {
  id: string;
  description: string;
  done: boolean;
  scope: string;
  type: 'task' | 'note';
  parentId: string | null;
  planId: string;
  [key: string]: unknown;
};

/** A long-term top-level task described as `item ${id}`, unless overridden. */
export const item = (id: string, extra: Partial<TestItem> = {}): TestItem => ({
  id,
  description: `item ${id}`,
  done: false,
  scope: 'longterm',
  type: 'task',
  parentId: null,
  planId: 'plan-1',
  ...extra,
});

/** n top-level tasks named `${prefix}1`..`${prefix}n`. */
export const topLevel = (n: number, prefix = 't') =>
  Array.from({ length: n }, (_, i) => item(`${prefix}${i + 1}`));

/** A `createChecklistItem` that echoes back what it was asked for under a fresh id. */
export const echoCreate = async (...args: unknown[]) => {
  // Untyped params so it fits `api.createChecklistItem.mockImplementation`.
  const [text, , , opts = {}] = args as [
    string,
    string,
    string,
    { type?: 'task' | 'note'; parentId?: string }?,
  ];
  nextId += 1;
  return item(`new${nextId}`, {
    description: text,
    type: opts.type ?? 'task',
    parentId: opts.parentId ?? null,
  });
};

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

type RenderOptions = {
  /** Which view to open on; omitted, Todo opens on its default (the blocks). */
  view?: 'list' | 'grid';
  /** Props over the default `fixedTaskType="longterm" enableTypeFilter`. */
  props?: Partial<ComponentProps<typeof Todo>>;
  /** Text to wait for; defaults to the first item's description. */
  ready?: string;
};

/** Serves `items` from `fetchChecklist`, renders Todo and waits for it to settle. */
export async function renderTodo(items: unknown[] = [], options: RenderOptions = {}) {
  if (options.view) window.localStorage.setItem('checklistView', options.view);
  api.fetchChecklist.mockImplementation(async () => items);
  const Todo = (await import('@/components/Todo')).default;
  const view = render(<Todo fixedTaskType="longterm" enableTypeFilter {...options.props} />);
  const first = (items[0] as { description?: string } | undefined)?.description;
  await screen.findByText(options.ready ?? first ?? 'No items yet. Add one below!');
  return view;
}

// ---------------------------------------------------------------------------
// Queries
// ---------------------------------------------------------------------------

/** The list row for `item ${id}`. */
export const rowOf = (id: string) => screen.getByText(`item ${id}`).closest('li')!;

/** The guide-rail role a row draws, or null for none. */
export const railOf = (id: string) =>
  rowOf(id).querySelector('[data-guide-rail]')?.getAttribute('data-guide-rail') ?? null;

/** The list view's add bar. */
export const addInput = () =>
  screen.getByRole('textbox', { name: /^New (task|note)/ }) as HTMLInputElement;

/** The id the add bar is nested under (it names its target), or null at top level. */
export const nestedUnder = () =>
  addInput().getAttribute('aria-label')?.match(/under “item (\w+)”/)?.[1] ?? null;

export const addBarRail = () => document.querySelector('[data-guide-rail="add-bar"]');

export const counter = () => screen.getByTestId('page-counter');
export const nextPage = () => screen.getByRole('button', { name: 'Next page' });
export const prevPage = () => screen.getByRole('button', { name: 'Previous page' });

/** The list / blocks toggle, which sits in the "View" group (FS-FSFEC R3). */
export const viewButton = (name: string) =>
  within(screen.getByRole('group', { name: 'View' })).getByRole('button', { name });

/** The type filter's tab, not the view toggle's same-named "Checklist". */
export const typeFilter = (name: string) =>
  screen
    .getAllByRole('button', { name })
    .find((b) => !b.closest('[role="group"][aria-label="View"]'))!;

/** A milestone card in the grid. */
export const card = (id: string) =>
  document.querySelector(`[data-checklist-card="${id}"]`) as HTMLElement;

/** Ids of the cards the composer is tracing. */
export const traced = () =>
  Array.from(document.querySelectorAll('[data-traced]')).map((el) =>
    el.getAttribute('data-checklist-card')
  );

/** Rendered order, read off the one control every item carries in both views. */
export const order = () =>
  screen
    .getAllByRole('button', { name: /^Actions for / })
    .map((b) => b.getAttribute('aria-label')!.replace('Actions for ', ''));

/** The grid's new-milestone composer and its state attribute. */
export const composer = () =>
  document.querySelector('[data-milestone-composer]') as HTMLElement;
export const composerState = () =>
  composer()?.getAttribute('data-milestone-composer') ?? null;

/** Opens an item's actions menu (list row or card) and picks `label` from it. */
export const chooseAction = (description: string, label: string | RegExp) => {
  fireEvent.click(screen.getByRole('button', { name: `Actions for ${description}` }));
  fireEvent.click(screen.getByRole('menuitem', { name: label }));
};

// ---------------------------------------------------------------------------
// The grid's new-milestone composer (milestone-composer*.test.tsx)
// ---------------------------------------------------------------------------

type User = ReturnType<typeof import('@testing-library/user-event').default.setup>;

/** The composer's resting "+ New milestone" / "+ New note" button. */
export const restingButton = () => within(composer()).getByRole('button');

/** Naming's input, for a milestone or (under Notes / the note icon) a note. */
export const namingInput = () =>
  within(composer()).queryByRole('textbox', {
    name: /^(Name a new milestone|New note)$/,
  }) as HTMLInputElement | null;

/** Filling's input. Scoped: "Add a step to ‹title›" is also a card's own input. */
export const fillingInput = (title: string) =>
  within(composer()).queryByRole('textbox', {
    name: `Add a step to ${title}`,
  }) as HTMLInputElement | null;

/** Rest → naming → filling on a milestone called `title`. */
export async function startFilling(user: User, title = 'Launch') {
  await user.click(restingButton());
  await user.type(namingInput()!, `${title}{Enter}`);
  await screen.findByRole('textbox', { name: `Add a step to ${title}` });
}
