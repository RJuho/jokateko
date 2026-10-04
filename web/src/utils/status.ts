/**
 * Workflow status that counts as finished. Mirrors the backend
 * (model.Task.IsDone, complete_task, dependency unblocking) which hard-codes "done".
 */
export const DONE_STATUS = 'done'

export function isDoneStatus(status: string | undefined | null): boolean {
	return status === DONE_STATUS
}
