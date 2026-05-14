'use client';

import { useState } from 'react';
import { Note, NoteType, NotePriority } from '@/types/notes';
import { ChecklistItem } from '@/services/api';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import {
  AlertTriangle,
  Sparkles,
  Lightbulb,
  MessageSquare,
  Info,
  X,
  Eye,
  EyeOff,
  Edit2,
  Link2,
  CheckCircle,
  AlertCircle,
  XCircle,
  Zap,
  Clock,
  Tag,
} from 'lucide-react';

interface NoteCardProps {
  note: Note;
  onUpdate: (noteId: string, updates: Partial<Note>) => void;
  onDelete: (noteId: string) => void;
  checklistItems: ChecklistItem[];
  className?: string;
}

export function NoteCard({
  note,
  onUpdate,
  onDelete,
  checklistItems,
  className,
}: NoteCardProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const [showRelatedTasks, setShowRelatedTasks] = useState(false);

  const getTypeIcon = () => {
    switch (note.type) {
      case 'warning':
        return <AlertTriangle className="h-4 w-4" />;
      case 'insight':
        return <Lightbulb className="h-4 w-4" />;
      case 'suggestion':
        return <Sparkles className="h-4 w-4" />;
      case 'ai':
        return <Zap className="h-4 w-4" />;
      case 'user':
        return <MessageSquare className="h-4 w-4" />;
      default:
        return <Info className="h-4 w-4" />;
    }
  };

  const getPriorityIcon = () => {
    switch (note.priority) {
      case 'critical':
        return <XCircle className="h-4 w-4 text-red-400/70" />;
      case 'high':
        return <AlertCircle className="h-4 w-4 text-amber-400/70" />;
      case 'medium':
        return <Info className="h-4 w-4 text-blue-400/70" />;
      case 'low':
        return <CheckCircle className="h-4 w-4 text-gray-400/70" />;
    }
  };

  const getCardStyles = () => {
    const baseStyles = 'relative overflow-hidden transition-all duration-300 hover:shadow-md bg-white/5 dark:bg-gray-900/10 backdrop-blur-sm';

    switch (note.type) {
      case 'warning':
        return cn(
          baseStyles,
          'border-l-4 border-l-amber-600/50'
        );
      case 'insight':
        return cn(
          baseStyles,
          'border-l-4 border-l-blue-600/50'
        );
      case 'suggestion':
        return cn(
          baseStyles,
          'border-l-4 border-l-emerald-600/50'
        );
      case 'ai':
        return cn(
          baseStyles,
          'border-l-4 border-l-purple-600/50'
        );
      case 'user':
        return cn(
          baseStyles,
          'border-l-4 border-l-gray-600/50'
        );
      default:
        return baseStyles;
    }
  };

  const getRelatedTasks = () => {
    if (!note.relatedTaskIds || note.relatedTaskIds.length === 0) return [];
    return checklistItems.filter(task => note.relatedTaskIds.includes(task.id));
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return 'just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;
    return date.toLocaleDateString();
  };

  const handleToggleRead = () => {
    onUpdate(note.id, { isRead: !note.isRead });
  };

  const handleDismiss = () => {
    if (note.type === 'warning' || note.type === 'suggestion') {
      onUpdate(note.id, { isDismissed: true });
      setTimeout(() => onDelete(note.id), 300);
    }
  };

  return (
    <Card
      className={cn(
        getCardStyles(),
        note.isDismissed && 'opacity-50',
        !note.isRead && 'ring-1 ring-gray-600',
        className
      )}
    >
      <div className="p-3">
        {/* Header */}
        <div className="flex items-start justify-between mb-2">
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-1">
              {getTypeIcon()}
              <span className="text-xs font-medium capitalize">
                {note.type}
              </span>
            </div>
            {note.aiMetadata && (
              <div className="relative group">
                <button
                  className="text-gray-400 hover:text-gray-300 transition-colors p-1"
                  aria-label="AI generation info"
                >
                  <Info className="w-3 h-3" />
                </button>
                <div className="absolute left-1/2 -translate-x-1/2 top-full mt-2 w-48 opacity-0 group-hover:opacity-100 transition-opacity duration-200 pointer-events-none z-[9999]">
                  <div className="bg-gray-900 border border-gray-700 rounded-lg p-2 shadow-xl">
                    <p className="text-xs text-white mb-1">{note.aiMetadata.sourceContext}</p>
                    <p className="text-xs text-gray-400">
                      Generated {formatDate(note.aiMetadata.generatedAt)}
                    </p>
                  </div>
                  <div className="absolute left-1/2 -translate-x-1/2 -top-1 w-2 h-2 bg-gray-900 border-l border-t border-gray-700 transform -rotate-45"></div>
                </div>
              </div>
            )}
          </div>
          <div className="flex items-center gap-1">
            <Button
              size="sm"
              variant="ghost"
              className="h-6 w-6 p-0"
              onClick={handleToggleRead}
            >
              {note.isRead ? (
                <EyeOff className="h-3 w-3" />
              ) : (
                <Eye className="h-3 w-3" />
              )}
            </Button>
            {(note.type === 'warning' || note.type === 'suggestion') && (
              <Button
                size="sm"
                variant="ghost"
                className="h-6 w-6 p-0"
                onClick={handleDismiss}
              >
                <X className="h-3 w-3" />
              </Button>
            )}
            {note.type === 'user' && (
              <Button
                size="sm"
                variant="ghost"
                className="h-6 w-6 p-0"
                onClick={() => onDelete(note.id)}
              >
                <X className="h-3 w-3" />
              </Button>
            )}
          </div>
        </div>

        {/* Content */}
        <div
          className={cn(
            'text-sm text-white mb-2',
            !isExpanded && 'line-clamp-2'
          )}
          onClick={() => setIsExpanded(!isExpanded)}
        >
          {note.content}
        </div>

        {/* Tags */}
        {note.tags && note.tags.length > 0 && (
          <div className="flex flex-wrap gap-1 mb-2">
            {note.tags?.map(tag => (
              <Badge key={tag} variant="outline" className="text-xs">
                <Tag className="h-2 w-2 mr-1" />
                {tag}
              </Badge>
            ))}
          </div>
        )}

        {/* Related Tasks */}
        {note.relatedTaskIds && note.relatedTaskIds.length > 0 && (
          <div className="mt-2">
            <Button
              size="sm"
              variant="ghost"
              className="h-6 text-xs p-1"
              onClick={() => setShowRelatedTasks(!showRelatedTasks)}
            >
              <Link2 className="h-3 w-3 mr-1" />
              {note.relatedTaskIds.length} Related Task{note.relatedTaskIds.length > 1 ? 's' : ''}
            </Button>

            {showRelatedTasks && (
              <div className="mt-2 space-y-1">
                {getRelatedTasks().map(task => (
                  <div
                    key={task.id}
                    className="text-xs p-1 rounded bg-white/5"
                  >
                    <span className={cn(
                      'inline-flex items-center gap-1 text-white',
                      task.done && 'line-through opacity-50'
                    )}>
                      {task.done ? (
                        <CheckCircle className="h-3 w-3 text-green-500" />
                      ) : (
                        <Clock className="h-3 w-3 text-gray-400" />
                      )}
                      {task.description}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

      </div>

      {/* Visual indicator for unread notes */}
      {!note.isRead && (
        <div className="absolute top-0 right-0 w-2 h-2 m-2 bg-gray-400 rounded-full animate-pulse" />
      )}

      {/* Priority indicator stripe */}
      {note.priority === 'critical' && (
        <div className="absolute top-0 left-0 right-0 h-0.5 bg-red-600/50" />
      )}
      {note.priority === 'high' && (
        <div className="absolute top-0 left-0 right-0 h-0.5 bg-amber-600/50" />
      )}
    </Card>
  );
}