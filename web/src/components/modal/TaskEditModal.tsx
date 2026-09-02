import { useEffect, useState } from 'preact/hooks'
import type { Priority, Task } from '../../schemas/models'
import {
	activeTaskDetailId,
	activeTaskEditId,
	config,
	configuredPriorities,
	milestones,
	mode,
	tasks,
	upsertTask,
} from '../../state/store'

export function TaskEditModal() {
	const taskId = activeTaskEditId.value
	const task = tasks.value.find((t) => t.id === taskId)
	const cols = config.value.board.columns
	const milestoneList = milestones.value
	const priorityList = configuredPriorities.value

	function closeModal() {
		activeTaskEditId.value = null
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

	if (!taskId || !task) {
		return null
	}

	const [title, setTitle] = useState(task.title)
	const [summary, setSummary] = useState(task.summary)
	const [status, setStatus] = useState(task.status)
	const [priority, setPriority] = useState<Priority>(task.priority)
	const [milestone, setMilestone] = useState(task.milestone || '')
	const [tagsStr, setTagsStr] = useState((task.tags || []).join(', '))
	const [depsStr, setDepsStr] = useState((task.dependencies || []).join(', '))
	const [body, setBody] = useState(task.body || '')
	const [isSaving, setIsSaving] = useState(false)
	const [error, setError] = useState<string | null>(null)

	async function handleSubmit(e: Event) {
		e.preventDefault()
		if (!task) {
			return
		}
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
			milestone: milestone.trim() === '' ? null : milestone.trim(),
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
			const updatedTask: Task = {
				...task,
				title: title.trim(),
				summary: summary.trim(),
				status,
				priority,
				milestone: milestone.trim() === '' ? null : milestone.trim(),
				tags: parsedTags,
				dependencies: parsedDeps,
				body,
				total_criteria: total,
				completed_criteria: completed,
			}
			upsertTask(updatedTask)
			closeModal()
			activeTaskDetailId.value = updatedTask.id
			setIsSaving(false)
			return
		}

		try {
			const res = await fetch(`/api/tasks/${encodeURIComponent(task.id)}`, {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(payload),
			})

			if (!res.ok) {
				const data = await res.json().catch(() => ({}))
				throw new Error(data.error || `HTTP ${res.status}`)
			}

			const updatedTask: Task = await res.json()
			upsertTask(updatedTask)
			closeModal()
			activeTaskDetailId.value = updatedTask.id
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
			aria-label={`Edit Task: ${task.title}`}
			data-testid='task-edit-modal'
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
					<h2 class='text-lg font-bold text-base-content'>Edit Task</h2>
					<button
						type='button'
						onClick={closeModal}
						class='btn btn-sm btn-ghost btn-circle'
						aria-label='Close edit modal'
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
							for='task-edit-title'
							class='label py-1 text-xs font-semibold'
						>
							Title <span class='text-error'>*</span>
						</label>
						<input
							id='task-edit-title'
							type='text'
							value={title}
							onInput={(e) => setTitle((e.target as HTMLInputElement).value)}
							class='input input-sm input-bordered w-full rounded-lg'
							required
							aria-label='Task title'
						/>
					</div>

					{/* Summary */}
					<div>
						<label
							for='task-edit-summary'
							class='label py-1 text-xs font-semibold'
						>
							Summary <span class='text-error'>*</span>
						</label>
						<textarea
							id='task-edit-summary'
							value={summary}
							onInput={(e) =>
								setSummary((e.target as HTMLTextAreaElement).value)
							}
							class='textarea textarea-sm textarea-bordered w-full rounded-lg'
							rows={2}
							required
							aria-label='Task summary'
						/>
					</div>

					{/* Column, Priority & Milestone Row */}
					<div class='grid grid-cols-1 sm:grid-cols-3 gap-2.5'>
						<div>
							<label
								for='task-edit-status'
								class='label py-1 text-xs font-semibold'
							>
								Column
							</label>
							<select
								id='task-edit-status'
								value={status}
								onChange={(e) =>
									setStatus((e.target as HTMLSelectElement).value)
								}
								class='select select-sm select-bordered w-full rounded-lg'
								aria-label='Select column'
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
								for='task-edit-priority'
								class='label py-1 text-xs font-semibold flex items-center justify-between'
							>
								<span>Priority</span>
								{priorityList.find((p) => p.id === priority)?.color && (
									<span
										class='w-2.5 h-2.5 rounded-full inline-block shadow-2xs'
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
								id='task-edit-priority'
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
								aria-label='Select priority'
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
								for='task-edit-milestone'
								class='label py-1 text-xs font-semibold'
							>
								Milestone
							</label>
							<select
								id='task-edit-milestone'
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
					</div>

					{/* Tags & Dependencies Row */}
					<div class='grid grid-cols-1 sm:grid-cols-2 gap-2.5'>
						<div>
							<label
								for='task-edit-tags'
								class='label py-1 text-xs font-semibold'
							>
								Tags (comma separated)
							</label>
							<input
								id='task-edit-tags'
								type='text'
								value={tagsStr}
								onInput={(e) =>
									setTagsStr((e.target as HTMLInputElement).value)
								}
								placeholder='frontend, ui/ux, api'
								class='input input-sm input-bordered w-full rounded-lg'
								aria-label='Task tags'
							/>
						</div>

						<div>
							<label
								for='task-edit-deps'
								class='label py-1 text-xs font-semibold'
							>
								Dependencies (IDs comma separated)
							</label>
							<input
								id='task-edit-deps'
								type='text'
								value={depsStr}
								onInput={(e) =>
									setDepsStr((e.target as HTMLInputElement).value)
								}
								placeholder='260901-task-a, 260901-task-b'
								class='input input-sm input-bordered w-full rounded-lg font-mono text-xs'
								aria-label='Task dependencies'
							/>
						</div>
					</div>

					{/* Markdown Body */}
					<div>
						<label
							for='task-edit-body'
							class='label py-1 text-xs font-semibold'
						>
							Body & Acceptance Criteria (Markdown)
						</label>
						<textarea
							id='task-edit-body'
							value={body}
							onInput={(e) => setBody((e.target as HTMLTextAreaElement).value)}
							class='textarea textarea-sm textarea-bordered w-full rounded-lg font-mono text-xs'
							rows={6}
							placeholder='## Acceptance Criteria&#10;- [ ] Criterion 1&#10;- [ ] Criterion 2'
							aria-label='Task markdown body'
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
						aria-label='Cancel editing'
					>
						Cancel
					</button>
					<button
						type='submit'
						disabled={isSaving}
						class='btn btn-primary btn-sm'
						aria-label='Save changes'
					>
						{isSaving ? 'Saving...' : 'Save Changes'}
					</button>
				</div>
			</form>
		</div>
	)
}
