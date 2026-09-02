import { useState } from 'preact/hooks'
import type { Task } from '../../schemas/models'
import { columnTasks, config, mode, tasks, upsertTask } from '../../state/store'
import { Column } from './Column'

export function KanbanBoard() {
	const cols = config.value.board.columns
	const groupedTasks = columnTasks.value
	const isLive = mode.value === 'live'
	const [activeMobileColumn, setActiveMobileColumn] = useState<string>(
		cols[0]?.id || 'backlog',
	)

	function handleDragStart(e: DragEvent, task: Task) {
		if (e.dataTransfer) {
			e.dataTransfer.setData('text/plain', task.id)
			e.dataTransfer.effectAllowed = 'move'
		}
	}

	async function handleTaskDrop(taskId: string, targetStatus: string) {
		const task = tasks.value.find((t) => t.id === taskId)
		if (!task || task.status === targetStatus) {
			return
		}

		// Optimistic update
		const prevStatus = task.status
		upsertTask({ ...task, status: targetStatus })

		if (!isLive) {
			return
		}

		try {
			const res = await fetch(
				`/api/tasks/${encodeURIComponent(taskId)}/status`,
				{
					method: 'PUT',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ status: targetStatus }),
				},
			)
			if (!res.ok) {
				throw new Error(`HTTP ${res.status}`)
			}
		} catch (err) {
			console.error('Failed to update task status:', err)
			// Rollback on error
			upsertTask({ ...task, status: prevStatus })
		}
	}

	function scrollToColumn(colId: string) {
		setActiveMobileColumn(colId)
		const el = document.getElementById(`kanban-col-${colId}`)
		el?.scrollIntoView({
			behavior: 'smooth',
			inline: 'center',
			block: 'nearest',
		})
	}

	return (
		<div class='flex flex-col flex-1 min-h-0 w-full'>
			{/* Mobile-Only Column Navigation Pills (< md screens) */}
			<div class='flex md:hidden items-center gap-1.5 overflow-x-auto no-scrollbar px-3 sm:px-4 pt-2 pb-1 border-b border-base-200'>
				<span class='text-[11px] text-base-content/50 uppercase font-bold tracking-wider shrink-0 mr-1'>
					Column:
				</span>
				{cols.map((c) => {
					const isSelected = activeMobileColumn === c.id
					const count = groupedTasks[c.id]?.length || 0
					return (
						<button
							key={c.id}
							type='button'
							onClick={() => scrollToColumn(c.id)}
							class={`btn btn-xs rounded-lg gap-1 whitespace-nowrap ${
								isSelected ? 'btn-primary' : 'btn-ghost'
							}`}
							aria-pressed={isSelected}
							aria-label={`Scroll to column ${c.name} (${count} tasks)`}
						>
							<span
								class='w-2 h-2 rounded-full'
								style={{ backgroundColor: c.color }}
								aria-hidden='true'
							/>
							{c.name}
							<span class='badge badge-xs badge-neutral'>{count}</span>
						</button>
					)
				})}
			</div>

			{/* Board Columns: scroll-snap-type x mandatory for single-column mobile view & fixed desktop widths */}
			<main
				class='flex-1 overflow-x-auto snap-x snap-mandatory scroll-smooth flex gap-3 sm:gap-4 p-3 sm:p-4 md:p-6 min-h-0 items-start'
				aria-label='Kanban columns'
			>
				{cols.map((col) => (
					<div key={col.id} class='shrink-0 snap-center'>
						<Column
							column={col}
							tasks={groupedTasks[col.id] || []}
							onTaskDrop={handleTaskDrop}
							onDragStart={handleDragStart}
						/>
					</div>
				))}
			</main>
		</div>
	)
}
