'use client';

import { useState, useEffect } from 'react';
import { Note, TaskNoteRelation } from '@/types/notes';
import { ChecklistItem } from '@/services/api';
import { NotesService } from '@/services/notesService';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import {
  AlertTriangle,
  Sparkles,
  Lightbulb,
  MessageSquare,
  Link2,
  Info,
  ChevronRight,
} from 'lucide-react';

interface TaskNoteRelationsProps {
  task: ChecklistItem;
  planId: string;
  className?: string;
  position?: 'tooltip' | 'inline' | 'sidebar';
}

export function TaskNoteRelations({
  task,
  planId,
  className,
  position = 'tooltip',
}: TaskNoteRelationsProps) {
  const notesService = NotesService.getInstance();
  const [relatedNotes, setRelatedNotes] = useState<Note[]>([]);
  const [isVisible, setIsVisible] = useState(false);

  useEffect(() => {
    const notes = notesService.getNotesForTask(planId, task.id);
    setRelatedNotes(notes);
  }, [task.id, planId]);

  if (relatedNotes.length === 0) return null;

  const getTypeIcon = (type: Note['type']) => {
    switch (type) {
      case 'warning':
        return <AlertTriangle className="h-3 w-3" />;
      case 'insight':
        return <Lightbulb className="h-3 w-3" />;
      case 'suggestion':
        return <Sparkles className="h-3 w-3" />;
      case 'ai':
        return <Sparkles className="h-3 w-3" />;
      case 'user':
        return <MessageSquare className="h-3 w-3" />;
      default:
        return <Info className="h-3 w-3" />;
    }
  };

  const getPriorityColor = (priority: Note['priority']) => {
    switch (priority) {
      case 'critical':
        return 'text-red-500 border-red-500';
      case 'high':
        return 'text-orange-500 border-orange-500';
      case 'medium':
        return 'text-blue-500 border-blue-500';
      case 'low':
        return 'text-green-500 border-green-500';
    }
  };

  const getRelationshipText = (note: Note): string => {
    // Determine relationship based on note type and content
    if (note.type === 'warning') return 'warns about';
    if (note.type === 'suggestion') return 'suggests for';
    if (note.type === 'insight') return 'insight about';
    if (note.content.toLowerCase().includes('block')) return 'blocks';
    if (note.content.toLowerCase().includes('depend')) return 'depends on';
    return 'related to';
  };

  // Tooltip position variant
  if (position === 'tooltip') {
    return (
      <div
        className={cn('relative inline-flex items-center', className)}
        onMouseEnter={() => setIsVisible(true)}
        onMouseLeave={() => setIsVisible(false)}
      >
        <Badge
          variant="outline"
          className="cursor-pointer h-5 px-1 text-xs"
        >
          <Link2 className="h-3 w-3 mr-1" />
          {relatedNotes.length}
        </Badge>

        {isVisible && (
          <div className="absolute bottom-full left-0 mb-2 z-50 min-w-[280px]">
            <div className="bg-white dark:bg-gray-900 border rounded-lg shadow-lg p-3 space-y-2">
              <div className="text-xs font-semibold mb-2">Related Notes</div>
              {relatedNotes.map((note) => (
                <div
                  key={note.id}
                  className={cn(
                    'flex items-start gap-2 p-2 rounded-md',
                    'bg-gray-50 dark:bg-gray-800',
                    'border-l-2',
                    getPriorityColor(note.priority)
                  )}
                >
                  <div className="flex-shrink-0 mt-0.5">
                    {getTypeIcon(note.type)}
                  </div>
                  <div className="flex-1">
                    <div className="text-xs font-medium flex items-center gap-1">
                      <span className="capitalize">{note.type}</span>
                      <ChevronRight className="h-2 w-2" />
                      <span className="text-gray-500">{getRelationshipText(note)}</span>
                    </div>
                    <p className="text-xs text-gray-600 dark:text-gray-400 mt-1 line-clamp-2">
                      {note.content}
                    </p>
                    {note.tags && note.tags.length > 0 && (
                      <div className="flex gap-1 mt-1 flex-wrap">
                        {note.tags?.slice(0, 3).map(tag => (
                          <Badge key={tag} variant="secondary" className="text-xs h-4 px-1">
                            {tag}
                          </Badge>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
            <div className="absolute -bottom-1 left-4 w-2 h-2 bg-white dark:bg-gray-900 border-r border-b transform rotate-45" />
          </div>
        )}
      </div>
    );
  }

  // Inline position variant
  if (position === 'inline') {
    return (
      <div className={cn('space-y-2 mt-2', className)}>
        {relatedNotes.map((note) => (
          <div
            key={note.id}
            className={cn(
              'flex items-start gap-2 p-2 rounded-md text-sm',
              'bg-gray-50 dark:bg-gray-800',
              'border-l-2 transition-all hover:shadow-sm',
              getPriorityColor(note.priority)
            )}
          >
            <div className="flex-shrink-0 mt-0.5">
              {getTypeIcon(note.type)}
            </div>
            <div className="flex-1">
              <p className="text-xs">{note.content}</p>
            </div>
          </div>
        ))}
      </div>
    );
  }

  // Sidebar position variant
  return (
    <div className={cn('space-y-3', className)}>
      <h4 className="text-sm font-semibold flex items-center gap-2">
        <Link2 className="h-4 w-4" />
        Related Notes ({relatedNotes.length})
      </h4>
      <div className="space-y-2">
        {relatedNotes.map((note) => (
          <div
            key={note.id}
            className={cn(
              'p-3 rounded-lg',
              'bg-white dark:bg-gray-900',
              'border transition-all hover:shadow-md',
              'cursor-pointer'
            )}
          >
            <div className="flex items-start justify-between mb-2">
              <div className="flex items-center gap-2">
                {getTypeIcon(note.type)}
                <span className="text-xs font-medium capitalize">{note.type}</span>
                <Badge variant="outline" className={cn('text-xs h-5', getPriorityColor(note.priority))}>
                  {note.priority}
                </Badge>
              </div>
            </div>
            <p className="text-sm text-gray-700 dark:text-gray-300 line-clamp-3">
              {note.content}
            </p>
            {note.aiMetadata && (
              <div className="mt-2 text-xs text-gray-500">
                <span>AI Confidence: {Math.round(note.aiMetadata.confidence * 100)}%</span>
              </div>
            )}
            {note.tags && note.tags.length > 0 && (
              <div className="flex gap-1 mt-2 flex-wrap">
                {note.tags?.map(tag => (
                  <Badge key={tag} variant="secondary" className="text-xs">
                    {tag}
                  </Badge>
                ))}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}