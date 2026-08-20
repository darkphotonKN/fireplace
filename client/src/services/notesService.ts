import { Note, NoteType, NotePriority, CreateNoteRequest, UpdateNoteRequest, TaskNoteRelation } from '@/types/notes';
import { ChecklistItem } from './api';
import { ApiError } from '@/api/client';
import * as notesApi from '@/api/notes';

/**
 * Notes is SERIALIZED (FS-0004, I-0018): every call below goes through the
 * generated client in `@/api/notes`, never through raw fetch. The
 * `{statusCode, message, result}` envelope is gone, so responses are used
 * directly instead of being unwrapped from `.result`.
 *
 * The 404-tolerant signatures (`updateNote` -> null, `deleteNote` -> false) are
 * preserved: the generated client throws an ApiError, and that is translated
 * back here so callers keep the contract they already had.
 */
const isNotFound = (err: unknown): boolean =>
  err instanceof ApiError && err.status === 404;

export class NotesService {
  private static instance: NotesService;

  private constructor() {}

  static getInstance(): NotesService {
    if (!NotesService.instance) {
      NotesService.instance = new NotesService();
    }
    return NotesService.instance;
  }

  // API Operations — all through the generated client.
  async createNote(planId: string, request: CreateNoteRequest): Promise<Note> {
    return (await notesApi.createNote(planId, request)) as Note;
  }

  async loadNotes(planId: string): Promise<Note[]> {
    return (await notesApi.listNotes(planId)) as Note[];
  }

  async updateNote(planId: string, noteId: string, updates: UpdateNoteRequest): Promise<Note | null> {
    try {
      return (await notesApi.updateNote(planId, noteId, updates)) as Note;
    } catch (err) {
      if (isNotFound(err)) return null;
      throw err;
    }
  }

  async deleteNote(planId: string, noteId: string): Promise<boolean> {
    try {
      await notesApi.deleteNote(planId, noteId);
      return true;
    } catch (err) {
      if (isNotFound(err)) return false;
      throw err;
    }
  }

  // Tag Generation
  generateTagsFromContent(content: string): string[] {
    const tags: string[] = [];
    const lowerContent = content.toLowerCase();

    // Time-based tags
    if (lowerContent.includes('morning') || lowerContent.includes('am')) tags.push('morning');
    if (lowerContent.includes('evening') || lowerContent.includes('pm')) tags.push('evening');
    if (lowerContent.includes('today')) tags.push('today');
    if (lowerContent.includes('tomorrow')) tags.push('tomorrow');
    if (lowerContent.includes('deadline')) tags.push('deadline');

    // Priority tags
    if (lowerContent.includes('urgent') || lowerContent.includes('asap')) tags.push('urgent');
    if (lowerContent.includes('important')) tags.push('important');

    // Action tags
    if (lowerContent.includes('review')) tags.push('review');
    if (lowerContent.includes('test') || lowerContent.includes('testing')) tags.push('testing');
    if (lowerContent.includes('bug') || lowerContent.includes('fix')) tags.push('bug-fix');
    if (lowerContent.includes('feature')) tags.push('feature');
    if (lowerContent.includes('refactor')) tags.push('refactor');

    return [...new Set(tags)]; // Remove duplicates
  }

  // AI Note Generation via Backend API
  async generateContextualNotes(
    planId: string,
    tasks: ChecklistItem[],
    planFocus: string
  ): Promise<Note[]> {
    // requestType 'all' generates every kind, matching the previous behaviour.
    return (await notesApi.generateAINotes(planId, 'all')) as Note[];
  }

  // Filter notes
  filterNotes(notes: Note[], criteria: {
    tags?: string[];
    type?: NoteType;
    relatedTaskId?: string;
    priority?: NotePriority;
    isRead?: boolean;
  }): Note[] {
    return notes.filter(note => {
      if (criteria.tags && criteria.tags.length > 0) {
        if (!criteria.tags.some(tag => (note.tags ?? []).includes(tag))) return false;
      }
      if (criteria.type && note.type !== criteria.type) return false;
      if (criteria.relatedTaskId && !(note.relatedTaskIds ?? []).includes(criteria.relatedTaskId)) return false;
      if (criteria.priority && note.priority !== criteria.priority) return false;
      if (criteria.isRead !== undefined && note.isRead !== criteria.isRead) return false;

      return true;
    });
  }

  // Generate task-note relationships
  generateTaskNoteRelations(tasks: ChecklistItem[], notes: Note[]): TaskNoteRelation[] {
    const relations: TaskNoteRelation[] = [];

    notes.forEach(note => {
      (note.relatedTaskIds ?? []).forEach(taskId => {
        const task = tasks.find(t => t.id === taskId);
        if (task) {
          let relationshipType: TaskNoteRelation['relationshipType'] = 'suggestion_for';
          let strength = 0.5;

          // Determine relationship type based on note type and content
          if (note.type === 'warning') {
            relationshipType = 'warns_about';
            strength = 0.8;
          } else if (note.type === 'insight') {
            relationshipType = 'inspired_by';
            strength = 0.6;
          } else if (note.content.toLowerCase().includes('block') || note.content.toLowerCase().includes('wait')) {
            relationshipType = 'blocks';
            strength = 0.9;
          } else if (note.content.toLowerCase().includes('depend')) {
            relationshipType = 'depends_on';
            strength = 0.7;
          }

          relations.push({
            taskId,
            noteId: note.id,
            relationshipType,
            strength
          });
        }
      });
    });

    return relations;
  }
}