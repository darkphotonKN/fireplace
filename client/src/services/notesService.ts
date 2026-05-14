import { Note, NoteType, NotePriority, CreateNoteRequest, UpdateNoteRequest, TaskNoteRelation, WarningNote } from '@/types/notes';
import { ChecklistItem, authFetch } from './api';
import { config } from '@/config/environment';

const API_BASE_URL = config.apiBaseUrl;

export class NotesService {
  private static instance: NotesService;

  private constructor() {}

  static getInstance(): NotesService {
    if (!NotesService.instance) {
      NotesService.instance = new NotesService();
    }
    return NotesService.instance;
  }

  // API Operations
  async createNote(planId: string, request: CreateNoteRequest): Promise<Note> {
    const response = await authFetch(`${API_BASE_URL}/api/plans/${planId}/notes`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(request),
    });

    if (!response.ok) {
      throw new Error(`Failed to create note: ${response.statusText}`);
    }

    const data = await response.json();
    return data.result;
  }

  async loadNotes(planId: string): Promise<Note[]> {
    const response = await authFetch(`${API_BASE_URL}/api/plans/${planId}/notes`);

    if (!response.ok) {
      throw new Error(`Failed to fetch notes: ${response.statusText}`);
    }

    const data = await response.json();
    return data.result || [];
  }

  async updateNote(planId: string, noteId: string, updates: UpdateNoteRequest): Promise<Note | null> {
    const response = await authFetch(`${API_BASE_URL}/api/plans/${planId}/notes/${noteId}`, {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(updates),
    });

    if (!response.ok) {
      if (response.status === 404) {
        return null;
      }
      throw new Error(`Failed to update note: ${response.statusText}`);
    }

    const data = await response.json();
    return data.result;
  }

  async deleteNote(planId: string, noteId: string): Promise<boolean> {
    const response = await authFetch(`${API_BASE_URL}/api/plans/${planId}/notes/${noteId}`, {
      method: 'DELETE',
    });

    if (!response.ok) {
      if (response.status === 404) {
        return false;
      }
      throw new Error(`Failed to delete note: ${response.statusText}`);
    }

    return true;
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
    const response = await authFetch(`${API_BASE_URL}/api/plans/${planId}/notes/generate-ai`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        requestType: 'all' // Generate all types of notes
      }),
    });

    if (!response.ok) {
      throw new Error(`Failed to generate AI notes: ${response.statusText}`);
    }

    const data = await response.json();
    return data.result || [];
  }

  // Mock AI Note Generation (DEPRECATED - kept for backwards compatibility)
  private async generateAINote(
    planId: string,
    context: {
      tasks: ChecklistItem[];
      planFocus: string;
      requestType: 'suggestion' | 'warning' | 'insight';
    }
  ): Promise<Note> {
    // Simulate API delay
    await new Promise(resolve => setTimeout(resolve, 800 + Math.random() * 400));

    const { tasks, planFocus, requestType } = context;
    let content = '';
    let priority: NotePriority = 'medium';
    let type: NoteType = 'ai';
    let relatedTaskIds: string[] = [];
    let tags: string[] = [];

    switch (requestType) {
      case 'warning':
        const overdueTasks = tasks.filter(t => {
          if (t.done) return false;
          const due = t.dueDate ?? t.scheduledTime;
          return due && new Date(due) < new Date();
        });
        if (overdueTasks.length > 0) {
          content = `⚠️ You have ${overdueTasks.length} overdue task${overdueTasks.length > 1 ? 's' : ''}. Consider reprioritizing or breaking them down into smaller steps.`;
          priority = overdueTasks.length > 2 ? 'high' : 'medium';
          relatedTaskIds = overdueTasks.map(t => t.id);
          tags = ['overdue', 'warning'];
        } else if (tasks.filter(t => !t.done).length > 10) {
          content = `📝 You have a lot of pending tasks. Consider archiving completed items and focusing on your top 3-5 priorities for today.`;
          priority = 'medium';
          tags = ['productivity', 'focus'];
        } else {
          content = `✅ Your task list is well-managed. Keep up the great work on "${planFocus}"!`;
          priority = 'low';
          tags = ['positive', 'progress'];
        }
        type = 'warning';
        break;

      case 'insight':
        const completedToday = tasks.filter(t => t.done);
        const completionRate = tasks.length > 0 ? (completedToday.length / tasks.length) * 100 : 0;

        if (completionRate > 70) {
          content = `🎯 Excellent progress! You've completed ${Math.round(completionRate)}% of your tasks. Consider adding new challenges for ${planFocus}.`;
          priority = 'low';
          tags = ['achievement', 'progress'];
        } else if (completionRate > 40) {
          content = `📊 You're making steady progress on ${planFocus}. Focus on completing 2-3 more tasks to build momentum.`;
          priority = 'medium';
          tags = ['progress', 'momentum'];
        } else {
          content = `💡 Consider breaking down complex tasks in ${planFocus} into smaller, actionable steps to improve completion rate.`;
          priority = 'medium';
          tags = ['strategy', 'productivity'];
        }
        type = 'insight';
        break;

      case 'suggestion':
        const unscheduledTasks = tasks.filter(t => !t.done && !t.scheduledTime);
        if (unscheduledTasks.length > 0) {
          content = `📅 You have ${unscheduledTasks.length} unscheduled tasks. Consider scheduling them to better manage your time for ${planFocus}.`;
          relatedTaskIds = unscheduledTasks.slice(0, 3).map(t => t.id);
          tags = ['scheduling', 'planning'];
        } else {
          content = `🚀 All tasks are scheduled! Consider reviewing your long-term goals for ${planFocus} and adding new objectives.`;
          tags = ['goals', 'planning'];
        }
        type = 'suggestion';
        break;
    }

    const note: Note = {
      id: `ai_note_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
      content,
      type,
      tags,
      relatedTaskIds,
      planId,
      priority,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      aiMetadata: {
        generatedFrom: 'task_analysis',
        confidence: 0.75 + Math.random() * 0.2,
        sourceContext: `Generated from ${tasks.length} tasks in ${planFocus}`,
        generatedAt: new Date().toISOString()
      },
      isRead: false,
      isDismissed: false
    };

    // Save the generated note
    const notes = this.loadNotes(planId);
    notes.push(note);
    this.saveNotes(planId, notes);

    return note;
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
        if (!criteria.tags.some(tag => note.tags.includes(tag))) return false;
      }
      if (criteria.type && note.type !== criteria.type) return false;
      if (criteria.relatedTaskId && !note.relatedTaskIds.includes(criteria.relatedTaskId)) return false;
      if (criteria.priority && note.priority !== criteria.priority) return false;
      if (criteria.isRead !== undefined && note.isRead !== criteria.isRead) return false;

      return true;
    });
  }

  // Get related notes for a task
  getNotesForTask(planId: string, taskId: string): Note[] {
    const notes = this.loadNotes(planId);
    return notes.filter(note => note.relatedTaskIds.includes(taskId));
  }

  // Generate task-note relationships
  generateTaskNoteRelations(tasks: ChecklistItem[], notes: Note[]): TaskNoteRelation[] {
    const relations: TaskNoteRelation[] = [];

    notes.forEach(note => {
      note.relatedTaskIds.forEach(taskId => {
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

  // Clear all notes for a plan
  clearNotes(planId: string): void {
    this.saveNotes(planId, []);
  }
}