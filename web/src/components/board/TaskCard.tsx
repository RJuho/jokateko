import { useMemo, useState } from 'preact/hooks'
import { navigateTo } from '../../router'
import type { Task } from '../../schemas/models'
import { mode, tasks } from '../../state/store'
import { PriorityBadge, TagBadge, TargetDateBadge } from '../common/Badge'

interface TaskCardProps {
	task: Task
	onDragStart?: (e: DragEvent, task: Task) => void
}

export function TaskCard({ task, onDragStart }: TaskCardProps) {
	const isLive = mode.value === 'live'
	const [idCopied, setIdCopied] = useState(false)

	function handleCopyId(e: MouseEvent) {
		e.stopPropagation()
		navigator.clipboard.writeText(task.id)
		setIdCopied(true)
		setTimeout(() => setIdCopied(false), 1500)
	}

	// Calculate if task has unfinished dependencies
	const isBlocked = useMemo(() => {
		if (!task.dependencies || task.dependencies.length === 0) {
			return false
		}
		return task.dependencies.some((depId) => {
			const dep = tasks.value.find((t) => t.id === depId)
			return dep?.status !== 'done'
		})
	}, [task.dependencies, tasks.value])

	const hasCriteria =
		task.total_criteria !== undefined && task.total_criteria > 0
	const criteriaProgress = hasCriteria
		? `${task.completed_criteria ?? 0}/${task.total_criteria}`
		: null

	function handleClick() {
		navigateTo(`task/${task.id}`)
	}

	return (
		<article
			class={`w-full card bg-base-100 border border-base-200/80 shadow-xs hover:shadow-md hover:border-primary/40 transition-all rounded-xl p-3.5 flex flex-col gap-2.5 select-none ${
				isLive ? 'cursor-grab active:cursor-grabbing' : ''
			} ${isBlocked ? 'border-l-4 border-l-warning' : ''}`}
			data-testid={`task-card-${task.id}`}
			draggable={isLive}
			onDragStart={(e) => onDragStart?.(e, task)}
		>
			{/* Card Header: Priority, Blocked Indicator, Clickable Copy ID (no "#") */}
			<div class='flex items-center justify-between gap-2'>
				<PriorityBadge priority={task.priority} />
				<div class='flex items-center gap-1.5'>
					{isBlocked && (
						<span
							class='badge badge-xs badge-warning p-1'
							title={`Blocked by unfinished dependencies: ${task.dependencies.join(
								', ',
							)}`}
						>
							<svg
								xmlns='http://www.w3.org/2000/svg'
								class='size-3'
								fill='none'
								viewBox='0 0 24 24'
								stroke='currentColor'
								stroke-width='2'
								aria-hidden='true'
							>
								<title>Blocked</title>
								<circle cx='12' cy='12' r='9' />
								<path stroke-linecap='round' d='M5.636 5.636l12.728 12.728' />
							</svg>
						</span>
					)}

					{/* Task ID: clickable to copy without "#", width-locked to prevent layout shift */}
					<button
						type='button'
						onClick={handleCopyId}
						class='text-[10px] font-mono text-base-content/40 hover:text-base-content hover:bg-base-200/70 px-1 py-0.5 rounded tracking-tight transition-colors'
						title={idCopied ? 'Copied!' : 'Click to copy ID'}
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
			</div>

			{/* Clickable Card Body & Footer */}
			<button
				type='button'
				draggable={isLive}
				onDragStart={(e) => onDragStart?.(e, task)}
				onClick={handleClick}
				class='w-full text-left flex flex-col gap-2.5 cursor-pointer focus:outline-hidden group'
				aria-label={`Open task: ${task.title}, Priority: ${task.priority}`}
			>
				{/* Card Title */}
				<h4 class='line-clamp-2 text-sm/snug font-semibold text-base-content transition-colors group-hover:text-primary'>
					{task.title}
				</h4>

				{/* Card Summary */}
				{task.summary && (
					<p class='line-clamp-2 text-xs/relaxed text-base-content/70'>
						{task.summary}
					</p>
				)}

				{/* Card Footer: Criteria, Target Date, Milestone, Tags */}
				<div class='flex flex-wrap items-center justify-between gap-1.5 pt-1 border-t border-base-200/50 mt-auto text-xs w-full'>
					<div class='flex flex-wrap items-center gap-1.5'>
						{/* Checklist Progress */}
						{criteriaProgress && (
							<div class='flex items-center gap-1 text-[11px] text-base-content/60 font-medium'>
								<svg
									xmlns='http://www.w3.org/2000/svg'
									class='size-3.5 text-primary'
									fill='none'
									viewBox='0 0 24 24'
									stroke='currentColor'
									aria-hidden='true'
								>
									<title>Checklist progress</title>
									<path
										stroke-linecap='round'
										stroke-linejoin='round'
										stroke-width='2'
										d='M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z'
									/>
								</svg>
								<span>{criteriaProgress}</span>
							</div>
						)}

						{/* Target Date Badge */}
						{task.target_at && (
							<TargetDateBadge targetAt={task.target_at} status={task.status} />
						)}

						{/* Milestone Tag */}
						{task.milestone && (
							<span
								class='badge badge-xs badge-ghost text-[10px] text-base-content/60 truncate max-w-[120px]'
								title={`Milestone: ${task.milestone}`}
							>
								{task.milestone}
							</span>
						)}
					</div>

					{/* Tags */}
					{task.tags && task.tags.length > 0 && (
						<div class='flex flex-wrap items-center gap-1 ml-auto'>
							{task.tags.slice(0, 2).map((tag) => (
								<TagBadge key={tag} tag={tag} />
							))}
							{task.tags.length > 2 && (
								<span class='text-[10px] text-base-content/40 font-medium'>
									+{task.tags.length - 2}
								</span>
							)}
						</div>
					)}
				</div>
			</button>
		</article>
	)
}
