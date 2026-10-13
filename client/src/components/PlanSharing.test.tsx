import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, within, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ApiError } from '@/api/client';
import type { PlanMember } from '@/api/plans';

/**
 * The Sharing section of a plan's Settings (FS-12ZMF R28–R30). The owner adds
 * people by email and can remove them; a member sees who they're working with
 * and nothing to press. The list only changes once the server has answered.
 */

const listPlanMembers = vi.fn();
const addPlanMember = vi.fn();
const removePlanMember = vi.fn();
vi.mock('@/api/plans', () => ({
  listPlanMembers: (...a: unknown[]) => listPlanMembers(...a),
  addPlanMember: (...a: unknown[]) => addPlanMember(...a),
  removePlanMember: (...a: unknown[]) => removePlanMember(...a),
}));

import PlanSharing from './PlanSharing';

const person = (
  id: string,
  name: string,
  relation: PlanMember['relation'] = 'member',
): PlanMember => ({
  id,
  name,
  email: `${name.toLowerCase()}@example.com`,
  relation,
  since: '2026-10-01T00:00:00Z',
});

const kranti = person('u-1', 'Kranti', 'owner');
const nick = person('u-2', 'Nick');
const sam = person('u-3', 'Sam');

const refusal = (status: number, code?: string) =>
  new ApiError(code ? { code } as never : undefined, status);

/** The people list, read row by row in the order it is drawn. */
const rows = () => within(screen.getByRole('list', { name: /people on this plan/i })).getAllByRole('listitem');

beforeEach(() => {
  listPlanMembers.mockReset().mockResolvedValue([kranti, nick]);
  addPlanMember.mockReset();
  removePlanMember.mockReset();
});

describe('PlanSharing — as a member', () => {
  it('should list the people with the owner marked and offer no controls', async () => {
    render(<PlanSharing planId="plan-1" relation="member" />);

    await screen.findByText('Nick');
    expect(listPlanMembers).toHaveBeenCalledWith('plan-1');
    expect(within(rows()[0]).getByText('Kranti')).toBeInTheDocument();
    expect(within(rows()[0]).getByText(/owner/i)).toBeInTheDocument();
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /add/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /remove/i })).not.toBeInTheDocument();
  });
});

describe('PlanSharing — as the owner', () => {
  it('should offer an email field and Remove on member rows only, owner first', async () => {
    render(<PlanSharing planId="plan-1" relation="owner" />);

    await screen.findByText('Nick');
    expect(screen.getByRole('textbox', { name: /email/i })).toBeInTheDocument();
    const [first, second] = rows();
    expect(within(first).getByText('Kranti')).toBeInTheDocument();
    expect(within(first).getByText('kranti@example.com')).toBeInTheDocument();
    expect(within(first).queryByRole('button', { name: /remove/i })).not.toBeInTheDocument();
    expect(within(second).getByText('nick@example.com')).toBeInTheDocument();
    expect(within(second).getByRole('button', { name: 'Remove Nick' })).toBeInTheDocument();
  });

  it('should show the new person and clear the field once the add succeeds', async () => {
    addPlanMember.mockResolvedValue(sam);
    render(<PlanSharing planId="plan-1" relation="owner" />);
    await screen.findByText('Nick');

    const field = screen.getByRole('textbox', { name: /email/i });
    await userEvent.type(field, '  sam@example.com ');
    await userEvent.click(screen.getByRole('button', { name: /^add$/i }));

    await screen.findByText('Sam');
    expect(addPlanMember).toHaveBeenCalledWith('plan-1', 'sam@example.com');
    expect(rows().map((r) => within(r).getByText(/^(Kranti|Nick|Sam)$/).textContent)).toEqual([
      'Kranti',
      'Nick',
      'Sam',
    ]);
    expect(field).toHaveValue('');
  });

  it('should leave the list as it was when someone already on it is added again', async () => {
    addPlanMember.mockResolvedValue(nick);
    render(<PlanSharing planId="plan-1" relation="owner" />);
    await screen.findByText('Nick');

    await userEvent.type(screen.getByRole('textbox', { name: /email/i }), 'nick@example.com');
    await userEvent.click(screen.getByRole('button', { name: /^add$/i }));

    await waitFor(() => expect(screen.getByRole('textbox', { name: /email/i })).toHaveValue(''));
    expect(rows()).toHaveLength(2);
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it.each([
    ['no account uses the email', refusal(400, 'NO_ACCOUNT_FOR_EMAIL'), 'No Fireplace account uses that email.'],
    ['the email is the owner’s own', refusal(400, 'MEMBER_IS_OWNER'), "That's you. You already own this plan."],
    ['the email is malformed', refusal(422, 'VALIDATION_FAILED'), 'Enter an email address.'],
    ['anything else goes wrong', refusal(503, 'SERVICE_UNAVAILABLE'), 'That service is temporarily unavailable. Please try again in a moment.'],
    ['the call never reaches the server', new TypeError('Failed to fetch'), 'Something went wrong on our end. Please try again.'],
  ])('should say so in words when %s', async (_case, err, words) => {
    addPlanMember.mockRejectedValue(err);
    render(<PlanSharing planId="plan-1" relation="owner" />);
    await screen.findByText('Nick');

    const field = screen.getByRole('textbox', { name: /email/i });
    await userEvent.type(field, 'who@example.com');
    await userEvent.click(screen.getByRole('button', { name: /^add$/i }));

    expect(await screen.findByRole('alert')).toHaveTextContent(words);
    expect(field).toHaveValue('who@example.com');
    expect(rows()).toHaveLength(2);
  });

  it('should drop the row once the remove succeeds', async () => {
    removePlanMember.mockResolvedValue(undefined);
    render(<PlanSharing planId="plan-1" relation="owner" />);
    await screen.findByText('Nick');

    await userEvent.click(screen.getByRole('button', { name: 'Remove Nick' }));

    await waitFor(() => expect(screen.queryByText('Nick')).not.toBeInTheDocument());
    expect(removePlanMember).toHaveBeenCalledWith('plan-1', 'u-2');
    expect(rows()).toHaveLength(1);
  });

  it('should not drop the row before the server has answered', async () => {
    removePlanMember.mockReturnValue(new Promise(() => {}));
    render(<PlanSharing planId="plan-1" relation="owner" />);
    await screen.findByText('Nick');

    await userEvent.click(screen.getByRole('button', { name: 'Remove Nick' }));

    expect(screen.getByText('Nick')).toBeInTheDocument();
  });

  it('should keep the row and show the error when the remove fails', async () => {
    removePlanMember.mockRejectedValue(refusal(503, 'SERVICE_UNAVAILABLE'));
    render(<PlanSharing planId="plan-1" relation="owner" />);
    await screen.findByText('Nick');

    await userEvent.click(screen.getByRole('button', { name: 'Remove Nick' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'That service is temporarily unavailable. Please try again in a moment.',
    );
    expect(screen.getByText('Nick')).toBeInTheDocument();
  });
});

describe('PlanSharing — loading the people', () => {
  it('should say so when the list cannot be loaded', async () => {
    listPlanMembers.mockRejectedValue(refusal(503, 'SERVICE_UNAVAILABLE'));
    render(<PlanSharing planId="plan-1" relation="owner" />);

    expect(await screen.findByRole('alert')).toHaveTextContent(/temporarily unavailable/i);
  });
});
