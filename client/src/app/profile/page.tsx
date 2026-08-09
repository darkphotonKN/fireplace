'use client';

import { useState, useEffect, useRef, useCallback } from 'react';
import { getProfile, updateProfile, type UserProfile } from '@/services/api';
import { ApiError } from '@/api/profile';
import { messageFor, fieldFor } from '@/api/client';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Button } from '@/components/ui/button';
import { useToast } from '@/components/ui/use-toast';

type EditableField = 'name' | 'displayName' | 'bio';

interface FieldConfig {
  label: string;
  key: EditableField;
  placeholder: string;
  type: 'input' | 'textarea';
}

const fields: FieldConfig[] = [
  { label: 'Name', key: 'name', placeholder: 'Your name', type: 'input' },
  { label: 'Display Name', key: 'displayName', placeholder: 'How you want to be called', type: 'input' },
  { label: 'Bio', key: 'bio', placeholder: 'Tell us about yourself...', type: 'textarea' },
];

export default function ProfilePage() {
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editingField, setEditingField] = useState<EditableField | null>(null);
  const [editValue, setEditValue] = useState('');
  const [saving, setSaving] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const { toast } = useToast();

  useEffect(() => {
    getProfile()
      .then((res) => setProfile(res))
      .catch((err) => setError(err.message || 'Failed to load profile'))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (editingField) {
      if (editingField === 'bio') {
        textareaRef.current?.focus();
      } else {
        inputRef.current?.focus();
      }
    }
  }, [editingField]);

  const startEdit = (field: EditableField) => {
    if (!profile) return;
    setEditValue(profile[field] || '');
    setEditingField(field);
  };

  const cancelEdit = () => {
    setEditingField(null);
    setEditValue('');
  };

  const saveField = useCallback(async () => {
    if (!profile || !editingField || saving) return;

    const trimmed = editValue.trim();

    if (editingField === 'name' && trimmed === '') {
      toast({ title: 'Validation error', description: 'Name cannot be empty', variant: 'destructive' });
      return;
    }

    const currentValue = profile[editingField] || '';
    if (trimmed === currentValue) {
      cancelEdit();
      return;
    }

    setSaving(true);
    try {
      const res = await updateProfile({ [editingField]: trimmed });
      setProfile(res);
      toast({ title: 'Profile updated' });
      cancelEdit();
    } catch (err) {
      // Branch on the RFC 9457 domain code (ADR-0004), never on the message —
      // `detail` is prose and is allowed to change between releases.
      // errors[] is empty for downstream failures, so the CODE carries the
      // field-level precision: PROFILE_NAME_EMPTY is field-specific by itself.
      const problem = err instanceof ApiError ? err.problem : undefined;
      toast({
        title: fieldFor(problem) ? 'Check that field' : 'Save failed',
        description: messageFor(problem),
        variant: 'destructive',
      });
    } finally {
      setSaving(false);
    }
  }, [profile, editingField, editValue, saving, toast]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      cancelEdit();
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <div className="text-foreground/50 text-base">Loading profile...</div>
      </div>
    );
  }

  if (error || !profile) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <div className="text-red-400 text-base">{error || 'Failed to load profile'}</div>
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto py-12 px-6">
      <h1 className="text-2xl font-semibold mb-8">Profile</h1>

      <div className="space-y-6">
        {/* Email — read only */}
        <div>
          <label className="block text-sm font-medium text-foreground/40 uppercase tracking-wider mb-1">
            Email
          </label>
          <div className="text-foreground/70 text-base py-2 px-3 rounded-md bg-foreground/5">
            {profile.email}
          </div>
        </div>

        {/* Editable fields */}
        {fields.map((field) => (
          <div key={field.key}>
            <label className="block text-sm font-medium text-foreground/40 uppercase tracking-wider mb-1">
              {field.label}
            </label>

            {editingField === field.key ? (
              <div className="space-y-2">
                {field.type === 'textarea' ? (
                  <Textarea
                    ref={textareaRef}
                    value={editValue}
                    onChange={(e) => setEditValue(e.target.value)}
                    onKeyDown={handleKeyDown}
                    placeholder={field.placeholder}
                    disabled={saving}
                    rows={3}
                  />
                ) : (
                  <Input
                    ref={inputRef}
                    value={editValue}
                    onChange={(e) => setEditValue(e.target.value)}
                    onKeyDown={handleKeyDown}
                    placeholder={field.placeholder}
                    disabled={saving}
                    className="bg-[#e5e2d3] dark:bg-gray-800/50 border-[#d4c9b0] dark:border-gray-700 text-foreground focus-visible:ring-[rgb(247,111,83)]/40"
                  />
                )}
                <div className="flex items-center gap-2">
                  <Button size="sm" onClick={saveField} disabled={saving}>
                    {saving ? 'Saving...' : 'Save'}
                  </Button>
                  <Button size="sm" variant="ghost" onClick={cancelEdit} disabled={saving}>
                    Cancel
                  </Button>
                </div>
              </div>
            ) : (
              <button
                onClick={() => startEdit(field.key)}
                className="w-full text-left text-base py-2 px-3 rounded-md bg-foreground/5 hover:bg-foreground/10 transition-colors cursor-pointer"
              >
                {profile[field.key] ? (
                  <span className="text-foreground/80">{profile[field.key]}</span>
                ) : (
                  <span className="text-foreground/30 italic">{field.placeholder}</span>
                )}
              </button>
            )}
          </div>
        ))}

        {/* Member since */}
        <div className="pt-4 border-t border-foreground/10">
          <span className="text-sm text-foreground/30">
            Member since {new Date(profile.createdAt).toLocaleDateString('en-US', { year: 'numeric', month: 'long' })}
          </span>
        </div>
      </div>
    </div>
  );
}
