import { useEffect, useState } from 'preact/hooks'
import type { Priority, Task } from '../../schemas/models'
import {
	activeTaskDetailId,
	config,
	configuredPriorities,
	createTaskInitialColumnId,
	createTaskInitialTargetDate,
	isCreateTaskModalOpen,
	milestones,
	mode,
	upsertTask,
} from '../../state/store'

export function CreateTaskModal() {
	const isOpen = isCreateTaskModalOpen.value
	const cols = config.value.board.columns
	const milestoneList = milestones.value
	const priorityList = configuredPriorities.value

	function closeModal() {
		createTaskInitialColumnId.value = null
		createTaskInitialTargetDate.value = null
		isCreateTaskModalOpen.value = false
	}

	useEffect(() => {
		function handleKeyDown(e: KeyboardEvent) {
			if (e.key === 'Escape') {
				closeModal()
			}
		}
		window.addEventListener('keydown', handleKeyDown)
		return () => window.removeEventListener('keydown', handleKeyDown)
	}, [])

	if (!isOpen) {
		return null
	}

	const [title, setTitle] = useState('')
	const [summary, setSummary] = useState('')
	const [status, setStatus] = useState(
		createTaskInitialColumnId.value
			|| config.value.board.default_create_state
			|| cols[0]?.id
			|| 'backlog',
	)
	const [priority, setPriority] = useState<Priority>('medium')
	const [milestone, setMilestone] = useState('')
	const [targetAt, setTargetAt] = useState(
		createTaskInitialTargetDate.value || '',
	)
	const [tagsStr, setTagsStr] = useState('')
	const [depsStr, setDepsStr] = useState('')
	const [body, setBody] = useState('')
	const [isSaving, setIsSaving] = useState(false)
	const [error, setError] = useState<string | null>(null)

	async function handleSubmit(e: Event) {
		e.preventDefault()
		if (!title.trim()) {
			setError('Title is required')
			return
		}
		if (!summary.trim()) {
			setError('Summary is required')
			return
		}

		setIsSaving(true)
		setError(null)

		const parsedTags = tagsStr
			.split(',')
			.map((t) => t.trim())
			.filter(Boolean)

		const parsedDeps = depsStr
			.split(',')
			.map((d) => d.trim())
			.filter(Boolean)

		const payload = {
			title: title.trim(),
			summary: summary.trim(),
			status,
			priority,
			milestone: milestone.trim() === '' ? undefined : milestone.trim(),
			target_at: targetAt.trim() === '' ? undefined : targetAt.trim(),
			tags: parsedTags,
			dependencies: parsedDeps,
			body,
		}

		if (mode.value !== 'live') {
			const total = body
				.split('\n')
				.filter((l) => /^\s*-\s*\[[ xX]\]/.test(l)).length
			const completed = body
				.split('\n')
				.filter((l) => /^\s*-\s*\[[xX]\]/.test(l)).length
			const now = new Date().toISOString()
			const createdTask: Task = {
				id: `task-${Date.now().toString(36)}`,
				title: title.trim(),
				summary: summary.trim(),
				status,
				priority,
				milestone: milestone.trim() === '' ? undefined : milestone.trim(),
				target_at: targetAt.trim() === '' ? undefined : targetAt.trim(),
				tags: parsedTags,
				dependencies: parsedDeps,
				body,
				total_criteria: total,
				completed_criteria: completed,
				created_at: now,
				changed_at: now,
			}
			upsertTask(createdTask)
			closeModal()
			activeTaskDetailId.value = createdTask.id
			setIsSaving(false)
			return
		}

		try {
			const res = await fetch('/api/tasks', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(payload),
			})

			if (!res.ok) {
				const data = await res.json().catch(() => ({}))
				throw new Error(data.error || `HTTP ${res.status}`)
			}

			const createdTask: Task = await res.json()
			upsertTask(createdTask)
			closeModal()
			activeTaskDetailId.value = createdTask.id
		} catch (err) {
			setError(err instanceof Error ? err.message : String(err))
		} finally {
			setIsSaving(false)
		}
	}

	return (
		<div
			class='modal modal-open z-50 bg-neutral/40 backdrop-blur-xs flex items-end sm:items-center justify-center p-0 sm:p-4'
			role='dialog'
			aria-modal='true'
			aria-label='Create New Task'
			data-testid='create-task-modal'
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
			<form
				onSubmit={handleSubmit}
				class='modal-box w-full max-w-2xl max-h-[90vh] sm:rounded-2xl rounded-b-none p-5 sm:p-6 overflow-y-auto bg-base-100 shadow-2xl flex flex-col gap-4 border border-base-300'
			>
				{/* Top Bar */}
				<div class='flex items-center justify-between border-b border-base-200 pb-3'>
					<h2 class='text-lg font-bold text-base-content'>Create New Task</h2>
					<button
						type='button'
						onClick={closeModal}
						class='btn btn-sm btn-ghost btn-circle'
						aria-label='Close create modal'
					>
						✕
					</button>
				</div>

				{/* Error Alert */}
				{error && (
					<div class='alert alert-error text-xs py-2 shadow-xs' role='alert'>
						<span>{error}</span>
					</div>
				)}

				{/* Form Fields */}
				<div class='flex flex-col gap-3 text-xs'>
					{/* Title */}
					<div>
						<label
							for='task-create-title'
							class='label py-1 text-xs font-semibold'
						>
							Title <span class='text-error'>*</span>
						</label>
						<input
							id='task-create-title'
							type='text'
							value={title}
							onInput={(e) => setTitle((e.target as HTMLInputElement).value)}
							placeholder='e.g., Implement OAuth2 flow'
							class='input input-sm input-bordered w-full rounded-lg'
							required
							aria-label='New task title'
							data-testid='task-create-title-input'
						/>
					</div>

					{/* Summary */}
					<div>
						<label
							for='task-create-summary'
							class='label py-1 text-xs font-semibold'
						>
							Summary <span class='text-error'>*</span>
						</label>
						<textarea
							id='task-create-summary'
							value={summary}
							onInput={(e) =>
								setSummary((e.target as HTMLTextAreaElement).value)
							}
							placeholder='Concise summary of work and scope'
							class='textarea textarea-sm textarea-bordered w-full rounded-lg'
							rows={2}
							required
							aria-label='New task summary'
							data-testid='task-create-summary-input'
						/>
					</div>

					{/* Column, Priority, Milestone & Target Date Row */}
					<div class='grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-2.5'>
						<div>
							<label
								for='task-create-status'
								class='label py-1 text-xs font-semibold'
							>
								Column
							</label>
							<select
								id='task-create-status'
								value={status}
								onChange={(e) =>
									setStatus((e.target as HTMLSelectElement).value)
								}
								class='select select-sm select-bordered w-full rounded-lg'
								aria-label='Select initial column'
							>
								{cols.map((c) => (
									<option key={c.id} value={c.id}>
										{c.name}
									</option>
								))}
							</select>
						</div>

						<div>
							<label
								for='task-create-priority'
								class='label py-1 text-xs font-semibold flex items-center justify-between'
							>
								<span>Priority</span>
								{priorityList.find((p) => p.id === priority)?.color && (
									<span
										class='inline-block size-2.5 rounded-full shadow-2xs'
										style={{
											backgroundColor: priorityList.find(
												(p) => p.id === priority,
											)?.color,
										}}
										aria-hidden='true'
									/>
								)}
							</label>
							<select
								id='task-create-priority'
								value={priority}
								onChange={(e) =>
									setPriority((e.target as HTMLSelectElement).value as Priority)
								}
								class='select select-sm select-bordered w-full rounded-lg'
								style={
									priorityList.find((p) => p.id === priority)?.color
										? {
												borderLeftColor: priorityList.find(
													(p) => p.id === priority,
												)?.color,
												borderLeftWidth: '3px',
											}
										: undefined
								}
								aria-label='Select initial priority'
							>
								{priorityList.map((p) => (
									<option key={p.id} value={p.id}>
										{p.name}
									</option>
								))}
							</select>
						</div>

						<div>
							<label
								for='task-create-milestone'
								class='label py-1 text-xs font-semibold'
							>
								Milestone
							</label>
							<select
								id='task-create-milestone'
								value={milestone}
								onChange={(e) =>
									setMilestone((e.target as HTMLSelectElement).value)
								}
								class='select select-sm select-bordered w-full rounded-lg'
								aria-label='Select milestone'
							>
								<option value=''>None</option>
								{milestoneList.map((m) => (
									<option key={m.id} value={m.id}>
										{m.title}
									</option>
								))}
							</select>
						</div>

						<div>
							<label
								for='task-create-target-at'
								class='label py-1 text-xs font-semibold'
							>
								Target Date & Time
							</label>
							<input
								id='task-create-target-at'
								type='datetime-local'
								value={targetAt}
								onInput={(e) =>
									setTargetAt((e.target as HTMLInputElement).value)
								}
								class='input input-sm input-bordered w-full rounded-lg'
								aria-label='Select target date and time'
								data-testid='task-create-target-at-input'
							/>
						</div>
					</div>

					{/* Tags & Dependencies Row */}
					<div class='grid grid-cols-1 sm:grid-cols-2 gap-2.5'>
						<div>
							<label
								for='task-create-tags'
								class='label py-1 text-xs font-semibold'
							>
								Tags (comma separated)
							</label>
							<input
								id='task-create-tags'
								type='text'
								value={tagsStr}
								onInput={(e) =>
									setTagsStr((e.target as HTMLInputElement).value)
								}
								placeholder='frontend, ui/ux, api'
								class='input input-sm input-bordered w-full rounded-lg'
								aria-label='New task tags'
							/>
						</div>

						<div>
							<label
								for='task-create-deps'
								class='label py-1 text-xs font-semibold'
							>
								Dependencies (IDs comma separated)
							</label>
							<input
								id='task-create-deps'
								type='text'
								value={depsStr}
								onInput={(e) =>
									setDepsStr((e.target as HTMLInputElement).value)
								}
								placeholder='260901-task-a, 260901-task-b'
								class='input input-sm input-bordered w-full rounded-lg font-mono text-xs'
								aria-label='New task dependencies'
							/>
						</div>
					</div>

					{/* Markdown Body */}
					<div>
						<label
							for='task-create-body'
							class='label py-1 text-xs font-semibold'
						>
							Body & Acceptance Criteria (Markdown)
						</label>
						<textarea
							id='task-create-body'
							value={body}
							placeholder='Enter task description or acceptance criteria (Markdown)...'
							onInput={(e) => setBody((e.target as HTMLTextAreaElement).value)}
							class='textarea textarea-sm textarea-bordered w-full rounded-lg font-mono text-xs'
							rows={6}
							aria-label='New task markdown body'
						/>
					</div>
				</div>

				{/* Modal Action Buttons */}
				<div class='modal-action flex items-center justify-end gap-2 pt-3 border-t border-base-200'>
					<button
						type='button'
						onClick={closeModal}
						disabled={isSaving}
						class='btn btn-ghost btn-sm'
						aria-label='Cancel task creation'
						data-testid='create-task-cancel-button'
					>
						Cancel
					</button>
					<button
						type='submit'
						disabled={isSaving}
						class='btn btn-primary btn-sm'
						aria-label='Create task'
						data-testid='create-task-submit-button'
					>
						{isSaving ? 'Creating...' : 'Create Task'}
					</button>
				</div>
			</form>
		</div>
	)
}
