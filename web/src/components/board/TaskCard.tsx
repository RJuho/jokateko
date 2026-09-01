import { useMemo } from 'preact/hooks'
import type { Task } from '../../schemas/models'
import { activeTaskDetailId, mode, tasks } from '../../state/store'
import { PriorityBadge, TagBadge } from '../common/Badge'

interface TaskCardProps {
	task: Task
	onDragStart?: (e: DragEvent, task: Task) => void
}

export function TaskCard({ task, onDragStart }: TaskCardProps) {
	const isLive = mode.value === 'live'

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
		activeTaskDetailId.value = task.id
	}

	return (
		<button
			type='button'
			class={`w-full text-left card bg-base-100 border border-base-200/80 shadow-xs hover:shadow-md hover:border-primary/40 transition-all rounded-xl cursor-pointer p-3.5 flex flex-col gap-2.5 focus:outline-hidden focus:ring-2 focus:ring-primary/40 ${
				isBlocked ? 'border-l-4 border-l-warning' : ''
			}`}
			onClick={handleClick}
			draggable={isLive}
			onDragStart={(e) => onDragStart?.(e, task)}
			aria-label={`Task: ${task.title}, Priority: ${task.priority}`}
			data-testid={`task-card-${task.id}`}
		>
			{/* Card Header: Priority, ID, Blocked Indicator */}
			<div class='flex items-center justify-between gap-2'>
				<PriorityBadge priority={task.priority} />
				<div class='flex items-center gap-1.5'>
					{isBlocked && (
						<span
							class='badge badge-xs badge-warning gap-1 text-[10px] font-medium'
							title={`Blocked by unfinished dependencies: ${task.dependencies.join(
								', ',
							)}`}
						>
							Blocked
						</span>
					)}
					<span class='text-[10px] font-mono text-base-content/40 tracking-tight'>
						#{task.id}
					</span>
				</div>
			</div>

			{/* Card Title */}
			<h4 class='text-sm font-semibold text-base-content leading-snug line-clamp-2'>
				{task.title}
			</h4>

			{/* Card Summary */}
			{task.summary && (
				<p class='text-xs text-base-content/70 line-clamp-2 leading-relaxed'>
					{task.summary}
				</p>
			)}

			{/* Card Footer: Criteria, Tags */}
			<div class='flex flex-wrap items-center justify-between gap-1.5 pt-1 border-t border-base-200/50 mt-auto text-xs'>
				{/* Checklist Progress */}
				{criteriaProgress && (
					<div class='flex items-center gap-1 text-[11px] text-base-content/60 font-medium'>
						<svg
							xmlns='http://www.w3.org/2000/svg'
							class='h-3.5 w-3.5 text-primary'
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

				{/* Milestone Tag */}
				{task.milestone && (
					<span
						class='badge badge-xs badge-ghost text-[10px] text-base-content/60 truncate max-w-[120px]'
						title={`Milestone: ${task.milestone}`}
					>
						{task.milestone}
					</span>
				)}

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
	)
}
