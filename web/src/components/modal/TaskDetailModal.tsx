import { useEffect, useRef, useState } from 'preact/hooks'
import { navigateTo, navigateToCalendar } from '../../router'
import type { Task } from '../../schemas/models'
import {
	activeTab,
	activeTaskDetailId,
	activeTaskEditId,
	config,
	filters,
	mode,
	removeTask,
	tasks,
	upsertTask,
} from '../../state/store'
import {
	renderMermaidDiagrams,
	reRenderMermaidDiagrams,
} from '../../utils/mermaid'
import { PriorityBadge, TagBadge, TargetDateBadge } from '../common/Badge'

function formatDateTime(isoStr?: string) {
	if (!isoStr) return ''
	try {
		const d = new Date(isoStr)
		if (Number.isNaN(d.getTime())) return isoStr
		return d.toLocaleDateString(undefined, {
			year: 'numeric',
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit',
		})
	} catch {
		return isoStr
	}
}

export function TaskDetailModal() {
	const taskId = activeTaskDetailId.value
	const task = tasks.value.find((t) => t.id === taskId)
	const isLive = mode.value === 'live'
	const cols = config.value.board.columns
	const [isDeleting, setIsDeleting] = useState(false)
	const [idCopied, setIdCopied] = useState(false)
	const bodyRef = useRef<HTMLDivElement>(null)

	function closeModal() {
		activeTaskDetailId.value = null
		if (activeTab.value === 'calendar') {
			navigateToCalendar()
			return
		}
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
	const editableStates = config.value.board?.editable_states || ['backlog']
	const isEditable = editableStates.includes(task.status)

	const [isEditingBody, setIsEditingBody] = useState(false)
	const [editedBody, setEditedBody] = useState(task.body || '')
	const [isSavingBody, setIsSavingBody] = useState(false)
	const [saveBodyError, setSaveBodyError] = useState<string | null>(null)

	const [noteInput, setNoteInput] = useState('')
	const [isAddingNote, setIsAddingNote] = useState(false)
	const [addNoteError, setAddNoteError] = useState<string | null>(null)
	const [isMaximized, setIsMaximized] = useState(false)

	useEffect(() => {
		setEditedBody(task?.body || '')
		setIsEditingBody(false)
		setSaveBodyError(null)
		setNoteInput('')
		setAddNoteError(null)
	}, [task?.id, task?.status])

	async function handleSaveBody() {
		if (!task) return
		setIsSavingBody(true)
		setSaveBodyError(null)

		const total = editedBody
			.split('\n')
			.filter((l) => /^\s*[-*]\s+\[[ xX]\]/.test(l)).length
		const completed = editedBody
			.split('\n')
			.filter((l) => /^\s*[-*]\s+\[[xX]\]/.test(l)).length

		const updatedTask: Task = {
			...task,
			body: editedBody,
			total_criteria: total,
			completed_criteria: completed,
		}
		upsertTask(updatedTask)

		if (isLive) {
			try {
				const res = await fetch(`/api/tasks/${encodeURIComponent(task.id)}`, {
					method: 'PUT',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ body: editedBody }),
				})
				if (!res.ok) {
					const data = await res.json().catch(() => ({}))
					throw new Error(data.error || `HTTP ${res.status}`)
				}
				const serverTask = await res.json()
				upsertTask(serverTask)
				setIsEditingBody(false)
			} catch (err) {
				setSaveBodyError(err instanceof Error ? err.message : String(err))
			} finally {
				setIsSavingBody(false)
			}
		} else {
			setIsSavingBody(false)
			setIsEditingBody(false)
		}
	}

	async function handleAddNote() {
		if (!task || !noteInput.trim()) return
		setIsAddingNote(true)
		setAddNoteError(null)

		if (isLive) {
			try {
				const res = await fetch(
					`/api/tasks/${encodeURIComponent(task.id)}/notes`,
					{
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({ note: noteInput.trim() }),
					},
				)
				if (!res.ok) {
					const data = await res.json().catch(() => ({}))
					throw new Error(data.error || `HTTP ${res.status}`)
				}
				const serverTask = await res.json()
				upsertTask(serverTask)
				setNoteInput('')
			} catch (err) {
				setAddNoteError(err instanceof Error ? err.message : String(err))
			} finally {
				setIsAddingNote(false)
			}
		} else {
			const now = `${new Date()
				.toISOString()
				.replace('T', ' ')
				.substring(0, 16)} UTC`
			const noteBlock = `### [${now}]\n\n${noteInput.trim()}\n`
			let newBody = task.body || ''
			if (newBody.toLowerCase().includes('## notes')) {
				newBody = `${newBody.trimEnd()}\n\n${noteBlock}`
			} else {
				newBody = `${newBody.trimEnd()}\n\n## Notes\n\n${noteBlock}`
			}
			const total = newBody
				.split('\n')
				.filter((l) => /^\s*[-*]\s+\[[ xX]\]/.test(l)).length
			const completed = newBody
				.split('\n')
				.filter((l) => /^\s*[-*]\s+\[[xX]\]/.test(l)).length
			upsertTask({
				...task,
				body: newBody,
				total_criteria: total,
				completed_criteria: completed,
			})
			setNoteInput('')
			setIsAddingNote(false)
		}
	}

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

	async function toggleCheckbox(checkboxIndex: number, newChecked: boolean) {
		if (!task) return

		let count = 0
		const lines = (task.body || '').split('\n')
		let lineFound = false
		for (let i = 0; i < lines.length; i++) {
			if (/^\s*[-*]\s+\[[ xX]\]/.test(lines[i])) {
				count++
				if (count === checkboxIndex) {
					lines[i] = newChecked
						? lines[i].replace(/\[[ ]\]/, '[x]')
						: lines[i].replace(/\[[xX]\]/, '[ ]')
					lineFound = true
					break
				}
			}
		}
		if (!lineFound) return

		const newBody = lines.join('\n')
		const total = lines.filter((l) => /^\s*[-*]\s+\[[ xX]\]/.test(l)).length
		const completed = lines.filter((l) => /^\s*[-*]\s+\[[xX]\]/.test(l)).length

		const updatedTask: Task = {
			...task,
			body: newBody,
			total_criteria: total,
			completed_criteria: completed,
		}
		upsertTask(updatedTask)

		if (isLive) {
			try {
				const res = await fetch(`/api/tasks/${encodeURIComponent(task.id)}`, {
					method: 'PUT',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ body: newBody }),
				})
				if (res.ok) {
					const serverTask = await res.json()
					upsertTask(serverTask)
				}
			} catch (err) {
				console.warn('Failed to toggle checkbox on server:', err)
			}
		}
	}

	useEffect(() => {
		const el = bodyRef.current
		if (!el) return

		const handleClick = (e: MouseEvent) => {
			const target = e.target as HTMLElement | null
			if (!target) return
			const cb = target.closest(
				'input[type="checkbox"]',
			) as HTMLInputElement | null
			if (!cb) return

			const idxStr = cb.getAttribute('data-checkbox-index')
			if (!idxStr) return

			const idx = parseInt(idxStr, 10)
			if (idx > 0) {
				if (!isLive) {
					e.preventDefault()
					return
				}
				toggleCheckbox(idx, cb.checked)
			}
		}

		el.addEventListener('click', handleClick)
		return () => el.removeEventListener('click', handleClick)
	}, [isLive, task?.id, task?.body])

	// Render Mermaid diagrams within task body and re-render on theme change
	useEffect(() => {
		const el = bodyRef.current
		if (!el) return

		renderMermaidDiagrams(el)

		const observer = new MutationObserver((mutations) => {
			for (const mutation of mutations) {
				if (
					mutation.type === 'attributes'
					&& mutation.attributeName === 'data-theme'
				) {
					reRenderMermaidDiagrams(el)
				}
			}
		})

		observer.observe(document.documentElement, {
			attributes: true,
			attributeFilter: ['data-theme'],
		})

		return () => observer.disconnect()
	}, [task?.id, task?.body_html, isEditingBody])

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
			<div
				class={`modal-box w-full ${
					isMaximized
						? 'max-w-[96vw] sm:max-w-[96vw] h-[92vh] max-h-[92vh]'
						: 'max-w-2xl max-h-[90vh]'
				} sm:rounded-2xl rounded-b-none p-5 sm:p-6 overflow-y-auto bg-base-100 shadow-2xl flex flex-col gap-3.5 border border-base-300 transition-all duration-200`}
			>
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

							{/* Target Date Badge */}
							{task.target_at && (
								<TargetDateBadge
									targetAt={task.target_at}
									status={task.status}
								/>
							)}

							{/* 4. Task ID: clickable to copy, no "#", width-locked to prevent layout shift */}
							<button
								type='button'
								onClick={handleCopyId}
								class='font-mono text-xs text-base-content/50 hover:text-base-content hover:bg-base-200/60 px-1.5 py-0.5 rounded transition-colors ml-auto sm:ml-0'
								title={idCopied ? 'Copied to clipboard!' : 'Click to copy ID'}
								aria-label={`Copy task ID ${task.id}`}
							>
								<span class="inline-grid [grid-template-areas:'stack'] items-center justify-center">
									<span class='[grid-area:stack] invisible'>{task.id}</span>
									<span
										class={`[grid-area:stack] text-center ${
											idCopied ? 'text-success font-semibold' : ''
										}`}
									>
										{idCopied ? 'copied!' : task.id}
									</span>
								</span>
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

					{/* Top Right Action Buttons: Maximize / Restore and Close (✕) */}
					<div class='flex items-center gap-1 shrink-0'>
						<button
							type='button'
							onClick={() => setIsMaximized(!isMaximized)}
							class='btn btn-sm btn-ghost btn-circle text-base-content/60 hover:text-base-content'
							title={isMaximized ? 'Restore size' : 'Expand modal'}
							aria-label={isMaximized ? 'Restore size' : 'Maximize modal'}
							data-testid='task-modal-maximize-btn'
						>
							{isMaximized ? (
								<svg
									class='size-4'
									fill='none'
									viewBox='0 0 24 24'
									stroke='currentColor'
									stroke-width='2'
									aria-hidden='true'
								>
									<path
										stroke-linecap='round'
										stroke-linejoin='round'
										d='M9 9L4 4m0 0h5m-5 0v5m11 0V4m0 0h-5m5 0l-5 5m-6 6l-5 5m0 0h5m-5 0v-5m16 0v5m0 0h-5m5 0l-5-5'
									/>
								</svg>
							) : (
								<svg
									class='size-4'
									fill='none'
									viewBox='0 0 24 24'
									stroke='currentColor'
									stroke-width='2'
									aria-hidden='true'
								>
									<path
										stroke-linecap='round'
										stroke-linejoin='round'
										d='M4 8V4m0 0h4M4 4l5 5m11-5h-4m4 0v4m0 0l-5-5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4'
									/>
								</svg>
							)}
						</button>
						<button
							type='button'
							onClick={closeModal}
							class='btn btn-sm btn-ghost btn-circle text-base-content/60 hover:text-base-content'
							aria-label='Close task details'
							data-testid='modal-close-button'
						>
							✕
						</button>
					</div>
				</div>

				{/* Summary sits cleanly under top bar, no "Summary" label or box */}
				{task.summary && (
					<p class='text-sm/relaxed text-base-content/85'>{task.summary}</p>
				)}

				{/* Timestamps Metadata */}
				{(task.created_at || task.changed_at || task.target_at) && (
					<div
						class='flex flex-wrap items-center gap-x-4 gap-y-1 text-[11px] text-base-content/50 pt-0.5'
						data-testid='task-timestamps'
					>
						{task.created_at && (
							<span data-testid='task-created-at'>
								Created:{' '}
								<span class='font-mono text-base-content/70'>
									{formatDateTime(task.created_at)}
								</span>
							</span>
						)}
						{task.changed_at && (
							<span data-testid='task-changed-at'>
								Updated:{' '}
								<span class='font-mono text-base-content/70'>
									{formatDateTime(task.changed_at)}
								</span>
							</span>
						)}
						{task.target_at && (
							<span data-testid='task-target-at'>
								Target:{' '}
								<span class='font-mono text-base-content/70'>
									{formatDateTime(task.target_at)}
								</span>
							</span>
						)}
					</div>
				)}

				{/* The ONLY line in task modal: between summary and body text */}
				<hr class='border-base-200 my-1' />

				{/* Section Header with edit spec button if editable */}
				<div class='my-1 flex items-center justify-between'>
					<span class='text-xs font-semibold uppercase tracking-wider text-base-content/50'>
						Specification
					</span>
					{isEditable && (
						<button
							type='button'
							onClick={() => {
								if (isEditingBody) {
									setEditedBody(task.body || '')
								}
								setIsEditingBody(!isEditingBody)
								setSaveBodyError(null)
							}}
							class='btn btn-ghost btn-xs text-primary'
							aria-label={
								isEditingBody
									? 'Cancel editing specification'
									: 'Edit specification'
							}
						>
							{isEditingBody ? 'Cancel' : 'Edit Spec'}
						</button>
					)}
				</div>

				{/* Task Body Editor (when editing in editable state) */}
				{isEditable && isEditingBody ? (
					<div class='flex flex-col gap-2 my-2'>
						{saveBodyError && (
							<div class='alert alert-error text-xs py-1.5 px-3 rounded-lg'>
								<span>{saveBodyError}</span>
							</div>
						)}
						<textarea
							value={editedBody}
							onInput={(e) =>
								setEditedBody((e.target as HTMLTextAreaElement).value)
							}
							rows={8}
							class='textarea textarea-bordered textarea-sm w-full font-mono text-xs rounded-lg'
							placeholder='Task markdown body and acceptance criteria...'
							data-testid='task-body-editor'
							aria-label='Task specification markdown editor'
						/>
						<div class='flex justify-end gap-2'>
							<button
								type='button'
								onClick={() => {
									setEditedBody(task.body || '')
									setIsEditingBody(false)
									setSaveBodyError(null)
								}}
								class='btn btn-ghost btn-xs'
							>
								Cancel
							</button>
							<button
								type='button'
								onClick={handleSaveBody}
								disabled={isSavingBody}
								class='btn btn-primary btn-xs'
								data-testid='save-body-button'
							>
								{isSavingBody ? 'Saving...' : 'Save Spec'}
							</button>
						</div>
					</div>
				) : /* Rendered Body: Markdown with @tailwindcss/typography (prose) and interactive checkboxes */
				task.body_html ? (
					<div
						ref={bodyRef}
						class='prose prose-sm max-w-none text-base-content/90 prose-headings:text-base-content prose-headings:font-bold prose-p:text-base-content/85 prose-strong:text-base-content prose-code:text-primary prose-code:bg-base-200/60 prose-code:px-1 prose-code:py-0.5 prose-code:rounded prose-code:before:content-none prose-code:after:content-none prose-pre:bg-base-200 prose-pre:text-base-content'
						dangerouslySetInnerHTML={{ __html: task.body_html }}
					/>
				) : task.body && task.body.trim() !== '' ? (
					<div
						ref={bodyRef}
						class='prose prose-sm max-w-none text-base-content/90 prose-p:text-base-content/85'
					>
						<p class='whitespace-pre-wrap'>{task.body}</p>
					</div>
				) : (
					<p class='text-xs text-base-content/40 italic'>
						No body specification provided.
					</p>
				)}

				{/* Add Note Section (only visible when in non-editable state) */}
				{!isEditable && (
					<div
						class='flex flex-col gap-2 p-3 bg-base-200/40 rounded-xl border border-base-200 mt-2'
						data-testid='add-note-section'
					>
						<div class='flex items-center justify-between'>
							<span class='text-xs font-semibold uppercase tracking-wider text-base-content/60'>
								Add Note
							</span>
							<span class='text-[11px] text-base-content/40'>
								Markdown supported, use - [ ] for checklists
							</span>
						</div>
						{addNoteError && (
							<div class='alert alert-error text-xs py-1 px-2.5 rounded-lg'>
								<span>{addNoteError}</span>
							</div>
						)}
						<textarea
							value={noteInput}
							onInput={(e) =>
								setNoteInput((e.target as HTMLTextAreaElement).value)
							}
							rows={3}
							class='textarea textarea-bordered textarea-sm w-full font-mono text-xs rounded-lg'
							placeholder='Add progress notes or follow-up criteria (e.g. - [ ] Check edge cases)...'
							data-testid='add-note-input'
							aria-label='Add note text input'
						/>
						<div class='flex justify-end'>
							<button
								type='button'
								onClick={handleAddNote}
								disabled={isAddingNote || !noteInput.trim()}
								class='btn btn-primary btn-xs'
								data-testid='add-note-button'
							>
								{isAddingNote ? 'Adding...' : 'Add Note'}
							</button>
						</div>
					</div>
				)}

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
							data-testid='edit-task-button'
						>
							Edit Task
						</button>
					</div>
				)}
			</div>
		</div>
	)
}
