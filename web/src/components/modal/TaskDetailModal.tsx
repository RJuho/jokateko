import { useEffect, useMemo, useState } from 'preact/hooks'
import { navigateTo } from '../../router'
import type { Task } from '../../schemas/models'
import {
	activeTaskDetailId,
	activeTaskEditId,
	config,
	filters,
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
	const [idCopied, setIdCopied] = useState(false)

	function closeModal() {
		activeTaskDetailId.value = null
		const activeMilestone = filters.value.selectedMilestone
		if (activeMilestone) {
			navigateTo(`milestone/${activeMilestone}`)
		} else {
			navigateTo('board')
		}
	}

	// Exit modal when pressing ESC
	useEffect(() => {
		function handleKeyDown(e: KeyboardEvent) {
			if (e.key === 'Escape') {
				closeModal()
			}
		}
		window.addEventListener('keydown', handleKeyDown)
		return () => window.removeEventListener('keydown', handleKeyDown)
	}, [])

	if (!taskId || !task) {
		return null
	}

	const currentColumn = cols.find((c) => c.id === task.status)

	function openEditModal() {
		activeTaskEditId.value = task?.id ?? null
		activeTaskDetailId.value = null
	}

	function handleCopyId() {
		if (!task) return
		navigator.clipboard.writeText(task.id)
		setIdCopied(true)
		setTimeout(() => setIdCopied(false), 1500)
	}

	async function handleDelete() {
		if (
			!task
			|| !confirm(`Are you sure you want to delete task "${task.title}"?`)
		) {
			return
		}
		setIsDeleting(true)
		if (!isLive) {
			removeTask(task.id)
			closeModal()
			setIsDeleting(false)
			return
		}

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

	async function toggleCheckbox(lineIndex: number, currentChecked: boolean) {
		if (!task) return
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

		if (isLive) {
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
	}

	// Parse body for interactive checkboxes & markdown text
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
		/* Backdrop: Click outside exits modal */
		<div
			class='modal modal-open z-50 bg-neutral/40 backdrop-blur-xs flex items-end sm:items-center justify-center p-0 sm:p-4'
			role='dialog'
			aria-modal='true'
			aria-label={`Task Details: ${task.title}`}
			data-testid='task-detail-modal'
			onClick={(e) => {
				if (e.target === e.currentTarget) {
					closeModal()
				}
			}}
			onKeyDown={(e) => {
				if (e.key === 'Escape') {
					closeModal()
				}
			}}
		>
			<div class='modal-box w-full max-w-2xl max-h-[90vh] sm:rounded-2xl rounded-b-none p-5 sm:p-6 overflow-y-auto bg-base-100 shadow-2xl flex flex-col gap-3.5 border border-base-300'>
				{/* Modal Top Bar: Board title pill before priority, x/y done badge after priority, copyable ID without # */}
				{/* No dividing line between top bar and summary */}
				<div class='flex items-start justify-between gap-3 pb-1'>
					<div class='flex flex-col gap-2 min-w-0 flex-1'>
						{/* Top metadata row */}
						<div class='flex items-center gap-2 flex-wrap'>
							{/* 1. Board Title with solid colored badge (no border) before priority, same size as other badges */}
							<span
								class='badge badge-xs font-bold text-slate-900 gap-1 px-1.5 py-0.5 border-0 shadow-2xs select-none'
								style={{
									backgroundColor: currentColumn?.color || '#94a3b8',
								}}
								title={`Column: ${currentColumn?.name || task.status}`}
							>
								<span
									class='size-1.5 shrink-0 rounded-full bg-slate-900/30'
									aria-hidden='true'
								/>
								<span>{currentColumn?.name || task.status}</span>
							</span>

							{/* 2. Priority Badge (small xs) */}
							<PriorityBadge priority={task.priority} />

							{/* 3. "x / y done" badge right after priority (only if acceptance criteria exist) */}
							{task.total_criteria !== undefined && task.total_criteria > 0 && (
								<span
									class='badge badge-xs badge-outline text-[10px] font-mono text-base-content/70 px-1.5 py-0.5'
									title='Acceptance criteria progress'
								>
									{task.completed_criteria ?? 0} / {task.total_criteria} done
								</span>
							)}

							{/* Milestone badge: clickable to filter board by milestone */}
							{task.milestone && (
								<button
									type='button'
									onClick={() => navigateTo(`milestone/${task.milestone}`)}
									class='badge badge-xs badge-ghost hover:badge-primary text-[10px] cursor-pointer transition-all hover:shadow-2xs'
									title={`Filter board by milestone: ${task.milestone}`}
									aria-label={`Go to milestone ${task.milestone}`}
									data-testid='task-modal-milestone-badge'
								>
									{task.milestone}
								</button>
							)}

							{/* 4. Task ID: clickable to copy, no "#" */}
							<button
								type='button'
								onClick={handleCopyId}
								class='font-mono text-xs text-base-content/50 hover:text-base-content hover:bg-base-200/60 px-1.5 py-0.5 rounded transition-colors ml-auto sm:ml-0'
								title={idCopied ? 'Copied to clipboard!' : 'Click to copy ID'}
								aria-label={`Copy task ID ${task.id}`}
							>
								{idCopied ? 'copied!' : task.id}
							</button>
						</div>

						{/* Task Title */}
						<h2 class='text-lg/snug font-bold text-base-content sm:text-xl'>
							{task.title}
						</h2>

						{/* Tags under title, no "Tags:" label */}
						{task.tags && task.tags.length > 0 && (
							<div class='flex flex-wrap items-center gap-1 pt-0.5'>
								{task.tags.map((tag) => (
									<TagBadge key={tag} tag={tag} />
								))}
							</div>
						)}
					</div>

					{/* Top Right Close Button (✕) */}
					<button
						type='button'
						onClick={closeModal}
						class='btn btn-sm btn-ghost btn-circle shrink-0 text-base-content/60 hover:text-base-content'
						aria-label='Close task details'
					>
						✕
					</button>
				</div>

				{/* Summary sits cleanly under top bar, no "Summary" label or box */}
				{task.summary && (
					<p class='text-sm/relaxed text-base-content/85'>{task.summary}</p>
				)}

				{/* The ONLY line in task modal: between summary and body text */}
				<hr class='border-base-200 my-1' />

				{/* Task Body Text & Interactive Checklists (starts right after summary line, no extra label) */}
				<div class='text-sm font-sans space-y-2'>
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
								class='text-xs/relaxed whitespace-pre-wrap text-base-content/80'
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

				{/* Dependencies (if any) */}
				{task.dependencies && task.dependencies.length > 0 && (
					<div class='flex flex-col gap-1.5 border border-base-200 p-3 rounded-xl bg-base-200/20 mt-2'>
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

				{/* Modal Actions: No "Close" button at the end */}
				{isLive && (
					<div class='modal-action flex items-center justify-between pt-2 border-t border-base-200 mt-2'>
						<button
							type='button'
							onClick={handleDelete}
							disabled={isDeleting}
							class='btn btn-error btn-outline btn-sm'
							aria-label='Delete this task'
						>
							{isDeleting ? 'Deleting...' : 'Delete'}
						</button>

						<button
							type='button'
							onClick={openEditModal}
							class='btn btn-primary btn-sm'
							aria-label='Edit this task'
						>
							Edit Task
						</button>
					</div>
				)}
			</div>
		</div>
	)
}
