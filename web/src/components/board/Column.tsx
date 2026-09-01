import { useState } from 'preact/hooks'
import type { Column as ColumnType, Task } from '../../schemas/models'
import { mode } from '../../state/store'
import { TaskCard } from './TaskCard'

interface ColumnProps {
	column: ColumnType
	tasks: Task[]
	onTaskDrop?: (taskId: string, targetStatus: string) => void
	onDragStart?: (e: DragEvent, task: Task) => void
}

export function Column({
	column,
	tasks,
	onTaskDrop,
	onDragStart,
}: ColumnProps) {
	const [isDragOver, setIsDragOver] = useState(false)
	const isLive = mode.value === 'live'

	function handleDragOver(e: DragEvent) {
		if (!isLive) {
			return
		}
		e.preventDefault()
		if (e.dataTransfer) {
			e.dataTransfer.dropEffect = 'move'
		}
		setIsDragOver(true)
	}

	function handleDragLeave() {
		setIsDragOver(false)
	}

	function handleDrop(e: DragEvent) {
		if (!isLive) {
			return
		}
		e.preventDefault()
		setIsDragOver(false)
		const taskId = e.dataTransfer?.getData('text/plain')
		if (taskId) {
			onTaskDrop?.(taskId, column.id)
		}
	}

	return (
		<section
			class={`flex flex-col bg-base-200/40 rounded-2xl border border-base-200 p-3 min-w-[280px] max-w-[320px] md:min-w-[300px] md:max-w-[340px] shrink-0 snap-center transition-all ${
				isDragOver ? 'ring-2 ring-primary/40 bg-primary/5' : ''
			}`}
			onDragOver={handleDragOver}
			onDragLeave={handleDragLeave}
			onDrop={handleDrop}
			aria-label={`Kanban column: ${column.name}`}
			data-testid={`column-${column.id}`}
		>
			{/* Column Header */}
			<div class='flex items-center justify-between pb-3 mb-2 border-b border-base-200/60'>
				<div class='flex items-center gap-2 min-w-0'>
					<span
						class='w-2.5 h-2.5 rounded-full shrink-0'
						style={{ backgroundColor: column.color }}
						aria-hidden='true'
					/>
					<h3 class='text-sm font-bold text-base-content tracking-tight truncate'>
						{column.name}
					</h3>
				</div>
				<span
					class='badge badge-sm badge-neutral font-semibold text-xs px-2'
					data-testid='column-task-count'
				>
					{tasks.length}
				</span>
			</div>

			{/* Task Cards List */}
			<ul
				class='flex flex-col gap-2.5 overflow-y-auto min-h-[150px] max-h-[calc(100vh-260px)] pr-0.5'
				aria-label={`Tasks in ${column.name}`}
			>
				{tasks.map((task) => (
					<li key={task.id} class='list-none'>
						<TaskCard task={task} onDragStart={onDragStart} />
					</li>
				))}

				{tasks.length === 0 && (
					<li class='list-none flex flex-col items-center justify-center py-8 text-center border-2 border-dashed border-base-300/60 rounded-xl text-base-content/40'>
						<span class='text-xs font-medium'>No tasks</span>
					</li>
				)}
			</ul>
		</section>
	)
}
