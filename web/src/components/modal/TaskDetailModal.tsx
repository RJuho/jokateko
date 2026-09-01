import { useMemo, useState } from 'preact/hooks'
import type { Task } from '../../schemas/models'
import {
	activeTaskDetailId,
	activeTaskEditId,
	config,
	mode,
	removeTask,
	tasks,
	upsertTask,
} from '../../state/store'
import { PriorityBadge, TagBadge } from '../common/Badge'

export function TaskDetailModal() {
	const taskId = activeTaskDetailId.value
	const task = tasks.value.find((t) => t.id === taskId)
	const isLive = mode.value === 'live'
	const cols = config.value.board.columns
	const [isDeleting, setIsDeleting] = useState(false)

	if (!taskId || !task) {
		return null
	}

	function closeModal() {
		activeTaskDetailId.value = null
	}

	function openEditModal() {
		activeTaskEditId.value = task?.id ?? null
		activeTaskDetailId.value = null
	}

	async function handleDelete() {
		if (
			!task
			|| !confirm(`Are you sure you want to delete task "${task.title}"?`)
		) {
			return
		}
		setIsDeleting(true)
		try {
			const res = await fetch(`/api/tasks/${encodeURIComponent(task.id)}`, {
				method: 'DELETE',
			})
			if (!res.ok) {
				const data = await res.json().catch(() => ({}))
				throw new Error(data.error || `HTTP ${res.status}`)
			}
			removeTask(task.id)
			closeModal()
		} catch (err) {
			alert(
				`Failed to delete task: ${err instanceof Error ? err.message : String(err)}`,
			)
		} finally {
			setIsDeleting(false)
		}
	}

	async function handleStatusChange(newStatus: string) {
		if (!task || task.status === newStatus) return
		const prev = task.status
		upsertTask({ ...task, status: newStatus })

		if (!isLive) return
		try {
			const res = await fetch(
				`/api/tasks/${encodeURIComponent(task.id)}/status`,
				{
					method: 'PUT',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ status: newStatus }),
				},
			)
			if (!res.ok) {
				upsertTask({ ...task, status: prev })
			}
		} catch (err) {
			console.warn('Failed to update status:', err)
			upsertTask({ ...task, status: prev })
		}
	}

	async function toggleCheckbox(lineIndex: number, currentChecked: boolean) {
		if (!task || !isLive) return
		const lines = (task.body || '').split('\n')
		const line = lines[lineIndex]
		if (!line) return

		const newLine = currentChecked
			? line.replace(/\[[xX]\]/, '[ ]')
			: line.replace(/\[ \]/, '[x]')

		lines[lineIndex] = newLine
		const newBody = lines.join('\n')

		const total = lines.filter((l) => /^\s*-\s*\[[ xX]\]/.test(l)).length
		const completed = lines.filter((l) => /^\s*-\s*\[[xX]\]/.test(l)).length

		const updatedTask: Task = {
			...task,
			body: newBody,
			total_criteria: total,
			completed_criteria: completed,
		}
		upsertTask(updatedTask)

		try {
			await fetch(`/api/tasks/${encodeURIComponent(task.id)}`, {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ body: newBody }),
			})
		} catch (err) {
			console.warn('Failed to toggle checkbox on server:', err)
		}
	}

	// Parse body for interactive checkboxes & text
	const bodyLines = useMemo(() => {
		return (task.body || '').split('\n').map((line, idx) => {
			const checkboxMatch = line.match(/^(\s*-\s*\[)([ xX])(\]\s*)(.*)$/)
			if (checkboxMatch) {
				const isChecked = checkboxMatch[2].toLowerCase() === 'x'
				const text = checkboxMatch[4]
				return {
					type: 'checkbox' as const,
					index: idx,
					checked: isChecked,
					text,
				}
			}
			return {
				type: 'text' as const,
				index: idx,
				text: line,
			}
		})
	}, [task.body])

	return (
		<div
			class='modal modal-open z-50 bg-neutral/40 backdrop-blur-xs flex items-end sm:items-center justify-center p-0 sm:p-4'
			role='dialog'
			aria-modal='true'
			aria-label={`Task Details: ${task.title}`}
			data-testid='task-detail-modal'
		>
			<div class='modal-box w-full max-w-2xl max-h-[90vh] sm:rounded-2xl rounded-b-none p-5 sm:p-6 overflow-y-auto bg-base-100 shadow-2xl flex flex-col gap-4 border border-base-300'>
				{/* Modal Top Bar */}
				<div class='flex items-start justify-between gap-3 border-b border-base-200 pb-3'>
					<div class='flex flex-col gap-1.5 min-w-0'>
						<div class='flex items-center gap-2 flex-wrap'>
							<PriorityBadge priority={task.priority} />
							<span class='font-mono text-xs text-base-content/50'>
								#{task.id}
							</span>
							{task.milestone && (
								<span class='badge badge-sm badge-ghost text-xs'>
									{task.milestone}
								</span>
							)}
						</div>
						<h2 class='text-lg sm:text-xl font-bold text-base-content leading-snug'>
							{task.title}
						</h2>
					</div>
					<button
						type='button'
						onClick={closeModal}
						class='btn btn-sm btn-ghost btn-circle shrink-0'
						aria-label='Close task details'
					>
						✕
					</button>
				</div>

				{/* Status Selector / Badge */}
				<div class='flex items-center justify-between gap-2 bg-base-200/50 p-3 rounded-xl'>
					<span class='text-xs font-semibold text-base-content/70 uppercase tracking-wider'>
						Column / Status:
					</span>
					{isLive ? (
						<select
							value={task.status}
							onChange={(e) =>
								handleStatusChange((e.target as HTMLSelectElement).value)
							}
							class='select select-sm select-bordered text-xs font-semibold rounded-lg'
							aria-label='Change task column status'
						>
							{cols.map((col) => (
								<option key={col.id} value={col.id}>
									{col.name}
								</option>
							))}
						</select>
					) : (
						<span class='badge badge-sm badge-neutral font-semibold uppercase'>
							{task.status}
						</span>
					)}
				</div>

				{/* Summary */}
				{task.summary && (
					<div class='bg-base-200/30 p-3.5 rounded-xl border border-base-200'>
						<h3 class='text-xs font-bold uppercase tracking-wider text-base-content/50 mb-1'>
							Summary
						</h3>
						<p class='text-sm text-base-content/85 leading-relaxed'>
							{task.summary}
						</p>
					</div>
				)}

				{/* Tags */}
				{task.tags && task.tags.length > 0 && (
					<div class='flex flex-wrap items-center gap-1.5'>
						<span class='text-xs font-semibold text-base-content/50 mr-1 uppercase'>
							Tags:
						</span>
						{task.tags.map((tag) => (
							<TagBadge key={tag} tag={tag} />
						))}
					</div>
				)}

				{/* Dependencies */}
				{task.dependencies && task.dependencies.length > 0 && (
					<div class='flex flex-col gap-1.5 border border-base-200 p-3 rounded-xl bg-base-200/20'>
						<h3 class='text-xs font-bold uppercase tracking-wider text-base-content/50'>
							Dependencies ({task.dependencies.length})
						</h3>
						<ul class='space-y-1.5'>
							{task.dependencies.map((depId) => {
								const dep = tasks.value.find((t) => t.id === depId)
								const isDone = dep?.status === 'done'
								return (
									<li
										key={depId}
										class='flex items-center justify-between gap-2 text-xs bg-base-100 p-2 rounded-lg border border-base-200'
									>
										<div class='flex items-center gap-2 min-w-0'>
											<span
												class={`w-2 h-2 rounded-full ${
													isDone ? 'bg-success' : 'bg-warning'
												}`}
												aria-hidden='true'
											/>
											<button
												type='button'
												onClick={() => {
													activeTaskDetailId.value = depId
												}}
												class='font-medium text-primary hover:underline truncate text-left'
												aria-label={`Open dependent task ${dep?.title || depId}`}
											>
												{dep ? dep.title : depId}
											</button>
										</div>
										<span class='badge badge-xs badge-ghost font-mono'>
											{dep ? dep.status : 'missing'}
										</span>
									</li>
								)
							})}
						</ul>
					</div>
				)}

				{/* Acceptance Criteria & Body */}
				<div class='flex flex-col gap-2 pt-1'>
					<div class='flex items-center justify-between'>
						<h3 class='text-xs font-bold uppercase tracking-wider text-base-content/50'>
							Spec & Acceptance Criteria
						</h3>
						{task.total_criteria !== undefined && task.total_criteria > 0 && (
							<span class='badge badge-sm badge-outline text-xs'>
								{task.completed_criteria ?? 0} / {task.total_criteria} done
							</span>
						)}
					</div>

					<div class='p-3.5 bg-base-200/30 rounded-xl border border-base-200 text-sm font-sans space-y-2'>
						{bodyLines.map((line) => {
							if (line.type === 'checkbox') {
								return (
									<label
										key={line.index}
										class={`flex items-start gap-2.5 p-1 rounded-md transition-colors cursor-pointer ${
											isLive ? 'hover:bg-base-200/60' : 'cursor-default'
										}`}
									>
										<input
											type='checkbox'
											checked={line.checked}
											disabled={!isLive}
											onChange={() => toggleCheckbox(line.index, line.checked)}
											class='checkbox checkbox-primary checkbox-xs mt-0.5'
											aria-label={`Mark criterion: ${line.text}`}
										/>
										<span
											class={`text-xs leading-relaxed ${
												line.checked
													? 'line-through text-base-content/40'
													: 'text-base-content'
											}`}
										>
											{line.text}
										</span>
									</label>
								)
							}

							if (line.text.startsWith('#')) {
								return (
									<div
										key={line.index}
										class='font-bold text-sm text-base-content pt-2'
									>
										{line.text.replace(/^#+\s*/, '')}
									</div>
								)
							}

							return (
								<p
									key={line.index}
									class='text-xs text-base-content/80 leading-relaxed whitespace-pre-wrap'
								>
									{line.text}
								</p>
							)
						})}

						{(!task.body || task.body.trim() === '') && (
							<p class='text-xs text-base-content/40 italic'>
								No body specification provided.
							</p>
						)}
					</div>
				</div>

				{/* Modal Actions */}
				<div class='modal-action flex items-center justify-between pt-2 border-t border-base-200 mt-2'>
					{isLive ? (
						<button
							type='button'
							onClick={handleDelete}
							disabled={isDeleting}
							class='btn btn-error btn-outline btn-sm'
							aria-label='Delete this task'
						>
							{isDeleting ? 'Deleting...' : 'Delete'}
						</button>
					) : (
						<div />
					)}

					<div class='flex items-center gap-2'>
						{isLive && (
							<button
								type='button'
								onClick={openEditModal}
								class='btn btn-primary btn-sm'
								aria-label='Edit this task'
							>
								Edit Task
							</button>
						)}
						<button
							type='button'
							onClick={closeModal}
							class='btn btn-ghost btn-sm'
							aria-label='Close modal'
						>
							Close
						</button>
					</div>
				</div>
			</div>
		</div>
	)
}
