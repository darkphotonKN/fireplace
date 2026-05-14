'use client';

import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { Note, NoteType, NotePriority, CreateNoteRequest } from '@/types/notes';
import { NotesService } from '@/services/notesService';
import { ChecklistItem } from '@/services/api';

interface AIContext {
  tasks: ChecklistItem[];
  planFocus: string;
  requestType: 'suggestion' | 'warning' | 'insight';
}

interface NotesContextType {
  notes: Note[];
  isGeneratingNote: boolean;
  addNote: (planId: string, note: CreateNoteRequest) => Note;
  updateNote: (planId: string, noteId: string, updates: Partial<Note>) => void;
  deleteNote: (planId: string, noteId: string) => void;
  generateAINote: (planId: string, context: AIContext) => Promise<Note>;
  getNotesForTask: (planId: string, taskId: string) => Note[];
  refreshNotes: (planId: string) => void;
  clearNotes: (planId: string) => void;
}

const NotesContext = createContext<NotesContextType | undefined>(undefined);

export function NotesProvider({ children }: { children: ReactNode }) {
  const notesService = NotesService.getInstance();
  const [notes, setNotes] = useState<Note[]>([]);
  const [isGeneratingNote, setIsGeneratingNote] = useState(false);
  const [currentPlanId, setCurrentPlanId] = useState<string | null>(null);

  const addNote = (planId: string, request: CreateNoteRequest): Note => {
    const newNote = notesService.createNote(planId, request);
    setNotes(prev => [...prev, newNote]);
    return newNote;
  };

  const updateNote = (planId: string, noteId: string, updates: Partial<Note>) => {
    const updatedNote = notesService.updateNote(planId, noteId, updates);
    if (updatedNote) {
      setNotes(prev => prev.map(n => n.id === noteId ? updatedNote : n));
    }
  };

  const deleteNote = (planId: string, noteId: string) => {
    if (notesService.deleteNote(planId, noteId)) {
      setNotes(prev => prev.filter(n => n.id !== noteId));
    }
  };

  const generateAINote = async (planId: string, context: AIContext): Promise<Note> => {
    setIsGeneratingNote(true);
    try {
      const generatedNote = await notesService.generateAINote(planId, context);
      setNotes(prev => [...prev, generatedNote]);
      return generatedNote;
    } finally {
      setIsGeneratingNote(false);
    }
  };

  const getNotesForTask = (planId: string, taskId: string): Note[] => {
    return notesService.getNotesForTask(planId, taskId);
  };

  const refreshNotes = (planId: string) => {
    const loadedNotes = notesService.loadNotes(planId);
    setNotes(loadedNotes);
    setCurrentPlanId(planId);
  };

  const clearNotes = (planId: string) => {
    notesService.clearNotes(planId);
    setNotes([]);
  };

  return (
    <NotesContext.Provider
      value={{
        notes,
        isGeneratingNote,
        addNote,
        updateNote,
        deleteNote,
        generateAINote,
        getNotesForTask,
        refreshNotes,
        clearNotes,
      }}
    >
      {children}
    </NotesContext.Provider>
  );
}

export function useNotes() {
  const context = useContext(NotesContext);
  if (!context) {
    throw new Error('useNotes must be used within a NotesProvider');
  }
  return context;
}