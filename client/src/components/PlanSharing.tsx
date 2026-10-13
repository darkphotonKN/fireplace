'use client';

import { FormEvent, useEffect, useState } from 'react';
import { ApiError, messageFor } from '@/api/client';
import {
  addPlanMember,
  listPlanMembers,
  removePlanMember,
  type PlanMember,
} from '@/api/plans';

/**
 * The Sharing section of a plan's Settings (FS-12ZMF R28–R30).
 *
 * Sharing here is two people keeping each other going on the same plan, not a
 * team being administered: there are no roles, no invitations and no cap. The
 * owner adds someone by the email they signed up with and can take them off
 * again. A member sees who owns the plan and who else is on it, and has
 * nothing to press.
 *
 * The list changes only once the server has answered (R29, R30). A successful
 * add returns the person's entry, so it is placed straight into the list (and
 * re-adding someone already there changes nothing); a successful remove drops
 * the row. Nothing is drawn optimistically.
 */

interface PlanSharingProps {
  planId: string;
  /** The viewer's relation to the plan, from getPlan. Never derived from ids. */
  relation: PlanMember['relation'];
}

/**
 * The add form's refusal in words (R29). The two sharing codes have their
 * own entries in the client's code map, so `messageFor` already says them;
 * 422 is the one refusal read from the status, because VALIDATION_FAILED's
 * general wording doesn't tell the owner what to fix here. `detail` is never
 * read.
 */
const addRefusal = (err: unknown): string => {
  if (err instanceof ApiError) {
    return err.status === 422 ? 'Enter an email address.' : err.message;
  }
  return messageFor(undefined);
};

const failure = (err: unknown): string =>
  err instanceof ApiError ? err.message : messageFor(undefined);

export default function PlanSharing({ planId, relation }: PlanSharingProps) {
  const isOwner = relation === 'owner';
  const [people, setPeople] = useState<PlanMember[] | null>(null);
  const [email, setEmail] = useState('');
  const [adding, setAdding] = useState(false);
  const [removingId, setRemovingId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let live = true;
    setError(null);
    listPlanMembers(planId)
      .then((list) => live && setPeople(list))
      .catch((err) => {
        console.error('Failed to load the people on this plan:', err);
        if (live) setError(failure(err));
      });
    return () => {
      live = false;
    };
  }, [planId]);

  const handleAdd = async (e: FormEvent) => {
    e.preventDefault();
    const wanted = email.trim();
    if (!wanted || adding) return;
    setAdding(true);
    setError(null);
    try {
      const entry = await addPlanMember(planId, wanted);
      setPeople((prev) => {
        const list = prev ?? [];
        return list.some((p) => p.id === entry.id) ? list : [...list, entry];
      });
      setEmail('');
    } catch (err) {
      setError(addRefusal(err));
    } finally {
      setAdding(false);
    }
  };

  const handleRemove = async (member: PlanMember) => {
    setRemovingId(member.id);
    setError(null);
    try {
      await removePlanMember(planId, member.id);
      setPeople((prev) => (prev ?? []).filter((p) => p.id !== member.id));
    } catch (err) {
      console.error('Failed to remove a person from this plan:', err);
      setError(failure(err));
    } finally {
      setRemovingId(null);
    }
  };

  const owner = people?.find((p) => p.relation === 'owner');
  const othersCount = people ? people.filter((p) => p.relation === 'member').length : 0;

  return (
    <section
      aria-labelledby="plan-sharing-heading"
      className="p-4 bg-white/5 dark:bg-gray-800/20 rounded-lg space-y-4"
    >
      <div>
        <h3 id="plan-sharing-heading" className="text-base font-medium mb-1">
          Sharing
        </h3>
        <p className="text-sm text-muted-foreground">
          {isOwner
            ? 'Bring someone along. Anyone you add can work in this plan with you, and you can keep each other going.'
            : owner
            ? `${owner.name} shared this plan with you. You're in it together.`
            : 'This plan was shared with you.'}
        </p>
      </div>

      {isOwner && (
        <form onSubmit={handleAdd} noValidate className="flex gap-2">
          <label htmlFor="plan-sharing-email" className="sr-only">
            Email of the person to add
          </label>
          <input
            id="plan-sharing-email"
            type="email"
            inputMode="email"
            autoComplete="off"
            placeholder="Their Fireplace email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            disabled={adding}
            className="flex-1 min-w-0 h-9 rounded-md border border-gray-300 dark:border-gray-600 bg-transparent px-3 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-orange-400"
          />
          <button
            type="submit"
            disabled={adding || !email.trim()}
            className="h-9 px-4 rounded-md text-sm font-medium bg-orange-500 text-white hover:bg-orange-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Add
          </button>
        </form>
      )}

      {error && (
        <p role="alert" className="text-sm text-red-600 dark:text-red-400">
          {error}
        </p>
      )}

      {people && (
        <ul aria-label="People on this plan" className="space-y-1">
          {people.map((p) => (
            <li
              key={p.id}
              className="flex items-center justify-between gap-3 rounded-md px-2 py-2 hover:bg-white/5 dark:hover:bg-gray-800/30"
            >
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium truncate">{p.name}</span>
                  {p.relation === 'owner' && (
                    <span className="text-xs rounded-full px-2 py-0.5 bg-orange-500/15 text-orange-600 dark:text-orange-300">
                      owner
                    </span>
                  )}
                </div>
                <span className="block text-xs text-muted-foreground truncate">
                  {p.email}
                </span>
              </div>
              {isOwner && p.relation === 'member' && (
                <button
                  type="button"
                  onClick={() => handleRemove(p)}
                  disabled={removingId === p.id}
                  aria-label={`Remove ${p.name}`}
                  className="shrink-0 text-xs text-muted-foreground hover:text-red-600 dark:hover:text-red-400 transition-colors disabled:opacity-50"
                >
                  Remove
                </button>
              )}
            </li>
          ))}
        </ul>
      )}

      {isOwner && people && othersCount === 0 && (
        <p className="text-sm text-muted-foreground">Just you for now.</p>
      )}
    </section>
  );
}
