import type { Task } from '../schemas/models'
import { tf } from './i18n'

/** Error code the server returns when a task is attached to an archived milestone. */
export const MILESTONE_ARCHIVED = 'milestone_archived'

/**
 * Sends a task create (POST) or update (PUT) and returns the saved task.
 * When the server refuses because the chosen milestone is closed or fully
 * completed, the user is asked to confirm, and the request is resent with
 * reopen_milestone. Any other failure throws with the server's message.
 */
export async function saveTask(
	url: string,
	method: 'POST' | 'PUT',
	payload: Record<string, unknown>,
	milestoneLabel: string,
): Promise<Task> {
	const send = async (body: Record<string, unknown>) => {
		const res = await fetch(url, {
			method,
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(body),
		})
		const data = await res.json().catch(() => ({}))
		return { res, data }
	}

	let { res, data } = await send(payload)
	if (
		res.status === 409
		&& data.code === MILESTONE_ARCHIVED
		&& confirm(
			tf('task_milestone_reopen_confirm', { milestone: milestoneLabel }),
		)
	) {
		;({ res, data } = await send({ ...payload, reopen_milestone: true }))
	}
	if (!res.ok) {
		throw new Error(data.error || `HTTP ${res.status}`)
	}
	return data as Task
}
