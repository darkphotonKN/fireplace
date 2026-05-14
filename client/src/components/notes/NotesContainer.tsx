'use client';

import { useState, useEffect, useCallback } from 'react';
import { NotesService } from '@/services/notesService';
import { Note, NoteType, NotePriority, NoteFilterType, NotesState } from '@/types/notes';
import { ChecklistItem } from '@/services/api';
import { NoteCard } from './NoteCard';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Card } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Badge } from '@/components/ui/badge';
import { useToast } from '@/components/ui/use-toast';
import { cn } from '@/lib/utils';
import {
  Sparkles,
  Plus,
  Filter,
  AlertTriangle,
  Lightbulb,
  MessageSquare,
  X,
  Loader2,
  RefreshCw,
  Tag,
} from 'lucide-react';

interface NotesContainerProps {
  planId: string;
  planFocus: string;
  checklistItems: ChecklistItem[];
  className?: string;
}

export function NotesContainer({
  planId,
  planFocus,
  checklistItems,
  className,
}: NotesContainerProps) {
  const { toast } = useToast();
  const notesService = NotesService.getInstance();

  const [state, setState] = useState<NotesState>({
    notes: [],
    isGeneratingNote: false,
    selectedTags: [],
    filterBy: 'all',
    activeNoteId: null,
  });

  const [isCreatingNote, setIsCreatingNote] = useState(false);
  const [newNoteContent, setNewNoteContent] = useState('');
  const [newNotePriority, setNewNotePriority] = useState<NotePriority>('medium');
  const [selectedTaskIds, setSelectedTaskIds] = useState<string[]>([]);

  // Load notes on mount
  useEffect(() => {
    const loadNotes = async () => {
      try {
        const loadedNotes = await notesService.loadNotes(planId);
        setState(prev => ({ ...prev, notes: loadedNotes }));
      } catch (error) {
        console.error('Failed to load notes:', error);
        toast({
          title: 'Failed to load notes',
          description: 'Could not retrieve your notes. Please try again.',
          variant: 'destructive',
        });
      }
    };
    loadNotes();
  }, [planId, toast]);

  // Get all unique tags
  const allTags = Array.from(
    new Set(state.notes.flatMap(note => note.tags))
  );

  // Filter notes based on current filter settings
  const filteredNotes = useCallback(() => {
    let notes = [...state.notes];

    // Filter by type
    if (state.filterBy !== 'all') {
      if (state.filterBy === 'ai') {
        notes = notes.filter(n => n.type === 'ai' || n.type === 'insight' || n.type === 'suggestion');
      } else if (state.filterBy === 'warnings') {
        notes = notes.filter(n => n.type === 'warning');
      } else if (state.filterBy === 'user') {
        notes = notes.filter(n => n.type === 'user');
      }
    }

    // Filter by tags
    if (state.selectedTags.length > 0) {
      notes = notes.filter(n =>
        state.selectedTags.some(tag => n.tags.includes(tag))
      );
    }

    // Sort by priority and date
    notes.sort((a, b) => {
      const priorityOrder = { critical: 0, high: 1, medium: 2, low: 3 };
      const priorityDiff = priorityOrder[a.priority] - priorityOrder[b.priority];
      if (priorityDiff !== 0) return priorityDiff;
      return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime();
    });

    return notes;
  }, [state.notes, state.filterBy, state.selectedTags]);

  // Create a new note
  const handleCreateNote = async () => {
    if (!newNoteContent.trim()) return;

    try {
      const tags = notesService.generateTagsFromContent(newNoteContent);
      const newNote = await notesService.createNote(planId, {
        content: newNoteContent,
        type: 'user',
        tags,
        relatedTaskIds: selectedTaskIds,
        priority: newNotePriority,
      });

      setState(prev => ({
        ...prev,
        notes: [...prev.notes, newNote],
      }));

      toast({
        title: 'Note created',
        description: 'Your note has been saved successfully.',
      });

      // Reset form
      setNewNoteContent('');
      setNewNotePriority('medium');
      setSelectedTaskIds([]);
      setIsCreatingNote(false);
    } catch (error) {
      console.error('Failed to create note:', error);
      toast({
        title: 'Failed to create note',
        description: 'Could not save your note. Please try again.',
        variant: 'destructive',
      });
    }
  };

  // Generate AI notes
  const handleGenerateAINotes = async () => {
    setState(prev => ({ ...prev, isGeneratingNote: true }));

    try {
      const generatedNotes = await notesService.generateContextualNotes(
        planId,
        checklistItems,
        planFocus
      );

      setState(prev => ({
        ...prev,
        notes: [...prev.notes, ...generatedNotes],
        isGeneratingNote: false,
      }));

      toast({
        title: 'AI Notes Generated',
        description: `Created ${generatedNotes.length} intelligent notes based on your tasks.`,
      });
    } catch (error) {
      setState(prev => ({ ...prev, isGeneratingNote: false }));
      toast({
        title: 'Generation Failed',
        description: 'Could not generate AI notes. Please try again.',
        variant: 'destructive',
      });
    }
  };

  // Update note
  const handleUpdateNote = async (noteId: string, updates: Partial<Note>) => {
    try {
      const updatedNote = await notesService.updateNote(planId, noteId, updates);
      if (updatedNote) {
        setState(prev => ({
          ...prev,
          notes: prev.notes.map(n => n.id === noteId ? updatedNote : n),
        }));
      }
    } catch (error) {
      console.error('Failed to update note:', error);
      toast({
        title: 'Failed to update note',
        description: 'Could not update the note. Please try again.',
        variant: 'destructive',
      });
    }
  };

  // Delete note
  const handleDeleteNote = async (noteId: string) => {
    try {
      const deleted = await notesService.deleteNote(planId, noteId);
      if (deleted) {
        setState(prev => ({
          ...prev,
          notes: prev.notes.filter(n => n.id !== noteId),
        }));

        toast({
          title: 'Note deleted',
          description: 'The note has been removed.',
        });
      }
    } catch (error) {
      console.error('Failed to delete note:', error);
      toast({
        title: 'Failed to delete note',
        description: 'Could not delete the note. Please try again.',
        variant: 'destructive',
      });
    }
  };

  // Toggle tag filter
  const toggleTagFilter = (tag: string) => {
    setState(prev => ({
      ...prev,
      selectedTags: prev.selectedTags.includes(tag)
        ? prev.selectedTags.filter(t => t !== tag)
        : [...prev.selectedTags, tag],
    }));
  };

  const getFilterIcon = (filter: NoteFilterType) => {
    switch (filter) {
      case 'ai': return <Sparkles className="h-4 w-4" />;
      case 'warnings': return <AlertTriangle className="h-4 w-4" />;
      case 'user': return <MessageSquare className="h-4 w-4" />;
      default: return <Filter className="h-4 w-4" />;
    }
  };

  const getFilterCount = (filter: NoteFilterType) => {
    if (filter === 'all') return state.notes.length;
    if (filter === 'ai') return state.notes.filter(n => n.type === 'ai' || n.type === 'insight' || n.type === 'suggestion').length;
    if (filter === 'warnings') return state.notes.filter(n => n.type === 'warning').length;
    if (filter === 'user') return state.notes.filter(n => n.type === 'user').length;
    return 0;
  };

  return (
    <Card className={cn('h-full flex flex-col', className)}>
      <div className="p-4 border-b">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold">Notes & Insights</h3>
          <div className="flex gap-2">
            <Button
              size="sm"
              variant="outline"
              onClick={() => setIsCreatingNote(!isCreatingNote)}
            >
              <Plus className="h-4 w-4 mr-1" />
              New Note
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={handleGenerateAINotes}
              disabled={state.isGeneratingNote}
            >
              {state.isGeneratingNote ? (
                <Loader2 className="h-4 w-4 mr-1 animate-spin" />
              ) : (
                <Sparkles className="h-4 w-4 mr-1" />
              )}
              AI Insights
            </Button>
          </div>
        </div>

        {/* Note Creation Form */}
        {isCreatingNote && (
          <div className="mb-4 p-4 bg-gray-900/50 backdrop-blur-sm rounded-lg space-y-3">
            <Textarea
              placeholder="Write your note..."
              value={newNoteContent}
              onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => setNewNoteContent(e.target.value)}
              className="min-h-[100px]"
            />
            <div className="flex items-center justify-between">
              <div className="flex gap-2">
                <select
                  value={newNotePriority}
                  onChange={(e) => setNewNotePriority(e.target.value as NotePriority)}
                  className="px-3 py-1 text-base border rounded-md"
                >
                  <option value="low">Low Priority</option>
                  <option value="medium">Medium Priority</option>
                  <option value="high">High Priority</option>
                  <option value="critical">Critical</option>
                </select>
              </div>
              <div className="flex gap-2">
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() => setIsCreatingNote(false)}
                >
                  Cancel
                </Button>
                <Button
                  size="sm"
                  onClick={handleCreateNote}
                  disabled={!newNoteContent.trim()}
                >
                  Create Note
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* Filter Tabs */}
        <Tabs
          value={state.filterBy}
          onValueChange={(value: string) => setState(prev => ({ ...prev, filterBy: value as NoteFilterType }))}
          className="w-full"
        >
          <TabsList className="grid grid-cols-4 w-full">
            <TabsTrigger value="all" className="flex items-center gap-1">
              {getFilterIcon('all')}
              All ({getFilterCount('all')})
            </TabsTrigger>
            <TabsTrigger value="ai" className="flex items-center gap-1">
              {getFilterIcon('ai')}
              AI ({getFilterCount('ai')})
            </TabsTrigger>
            <TabsTrigger value="warnings" className="flex items-center gap-1">
              {getFilterIcon('warnings')}
              Warnings ({getFilterCount('warnings')})
            </TabsTrigger>
            <TabsTrigger value="user" className="flex items-center gap-1">
              {getFilterIcon('user')}
              User ({getFilterCount('user')})
            </TabsTrigger>
          </TabsList>
        </Tabs>

        {/* Tag Filters */}
        {allTags.length > 0 && (
          <div className="mt-3 flex flex-wrap gap-1">
            {allTags.map(tag => (
              <Badge
                key={tag}
                variant={state.selectedTags.includes(tag) ? 'default' : 'outline'}
                className="cursor-pointer"
                onClick={() => toggleTagFilter(tag)}
              >
                <Tag className="h-3 w-3 mr-1" />
                {tag}
              </Badge>
            ))}
          </div>
        )}
      </div>

      {/* Notes List */}
      <ScrollArea className="flex-1">
        <div className="p-4 space-y-3">
          {filteredNotes().length > 0 ? (
            filteredNotes().map(note => (
              <NoteCard
                key={note.id}
                note={note}
                onUpdate={handleUpdateNote}
                onDelete={handleDeleteNote}
                checklistItems={checklistItems}
              />
            ))
          ) : (
            <div className="text-center py-8 text-gray-400">
              <Lightbulb className="h-12 w-12 mx-auto mb-3 opacity-30" />
              <p className="text-sm">No notes yet</p>
              <p className="text-xs mt-1 text-gray-500">Create a note or generate AI insights to get started</p>
            </div>
          )}
        </div>
      </ScrollArea>
    </Card>
  );
}