import { useState } from 'preact/hooks'
import type { Column as ColumnType, Task } from '../../schemas/models'
import {
	createTaskInitialColumnId,
	isCreateTaskModalOpen,
	mode,
} from '../../state/store'
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
			id={`kanban-col-${column.id}`}
			class={`flex flex-col bg-base-200/40 rounded-2xl border border-base-200 p-3 w-[calc(100vw-2.5rem)] sm:w-80 shrink-0 snap-center transition-all ${
				isDragOver ? 'ring-2 ring-primary/40 bg-primary/5' : ''
			}`}
			onDragOver={handleDragOver}
			onDragLeave={handleDragLeave}
			onDrop={handleDrop}
			aria-label={`Kanban column: ${column.name}`}
			data-testid={`column-${column.id}`}
		>
			{/* Column Header: Pill with solid board color, no border */}
			<div
				class='flex items-center justify-between px-2.5 py-1.5 rounded-full mb-2.5 shadow-xs select-none'
				style={{ backgroundColor: column.color }}
			>
				{/* Left: Circle with task count + Title */}
				<div class='flex items-center gap-2 min-w-0'>
					<span
						class='w-5 h-5 rounded-full flex items-center justify-center text-[11px] font-bold bg-base-100 text-base-content shadow-2xs shrink-0'
						data-testid='column-task-count'
						title={`${tasks.length} tasks in ${column.name}`}
					>
						{tasks.length}
					</span>
					<h3 class='text-xs sm:text-sm font-bold text-slate-900 tracking-tight truncate'>
						{column.name}
					</h3>
				</div>

				{/* Right: Plus Button to add a new task to this column */}
				{isLive && (
					<button
						type='button'
						onClick={() => {
							createTaskInitialColumnId.value = column.id
							isCreateTaskModalOpen.value = true
						}}
						class='w-5 h-5 rounded-full hover:bg-black/15 flex items-center justify-center text-slate-900/80 hover:text-slate-900 transition-colors shrink-0 ml-1.5'
						aria-label={`Add new task to ${column.name}`}
						data-testid={`add-task-${column.id}`}
					>
						<svg
							class='w-3.5 h-3.5'
							fill='none'
							viewBox='0 0 24 24'
							stroke='currentColor'
							stroke-width='2.5'
							aria-hidden='true'
						>
							<title>Add task</title>
							<path
								stroke-linecap='round'
								stroke-linejoin='round'
								d='M12 4v16m8-8H4'
							/>
						</svg>
					</button>
				)}
			</div>

			{/* Task Cards List: Fixed min-height so empty columns never jump or collapse */}
			<ul
				class='flex flex-col gap-2.5 overflow-y-auto min-h-[180px] max-h-[calc(100vh-230px)] pr-0.5'
				aria-label={`Tasks in ${column.name}`}
			>
				{tasks.map((task) => (
					<li key={task.id} class='list-none'>
						<TaskCard task={task} onDragStart={onDragStart} />
					</li>
				))}

				{tasks.length === 0 && (
					<li class='list-none flex flex-col items-center justify-center py-10 text-center border-2 border-dashed border-base-300/60 rounded-xl text-base-content/40 flex-1 min-h-[120px]'>
						<span class='text-xs font-medium'>No tasks</span>
					</li>
				)}
			</ul>
		</section>
	)
}
