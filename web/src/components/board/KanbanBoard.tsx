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
				const data = await res.json().catch(() => ({}))
				throw new Error(data.error || `HTTP ${res.status}`)
			}
		} catch (err) {
			console.warn('Failed to update task status on server:', err)
			// Rollback on error
			upsertTask({ ...task, status: prevStatus })
		}
	}

	return (
		<div class='flex flex-col flex-1 min-h-0 w-full'>
			{/* Mobile-Only Column Quick-Tab Pills (< md screens) */}
			<div class='flex md:hidden items-center gap-1.5 overflow-x-auto no-scrollbar px-4 pt-3 pb-1 border-b border-base-200'>
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
							onClick={() => setActiveMobileColumn(c.id)}
							class={`btn btn-xs rounded-lg gap-1 whitespace-nowrap ${
								isSelected ? 'btn-primary' : 'btn-ghost'
							}`}
							aria-pressed={isSelected}
							aria-label={`View column ${c.name} (${count} tasks)`}
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

			{/* Board Content */}
			{/* Desktop (md+): Multi-column horizontal scrolling */}
			{/* Mobile (< md): Displays active column or allows full swipe */}
			<main
				class='flex-1 overflow-x-auto snap-x snap-mandatory flex gap-4 p-4 md:p-6 min-h-0 items-start'
				aria-label='Kanban columns'
			>
				{cols.map((col) => {
					const isHiddenOnMobile =
						activeMobileColumn !== col.id ? 'hidden md:flex' : 'flex md:flex'

					return (
						<div key={col.id} class={`${isHiddenOnMobile} flex-col shrink-0`}>
							<Column
								column={col}
								tasks={groupedTasks[col.id] || []}
								onTaskDrop={handleTaskDrop}
								onDragStart={handleDragStart}
							/>
						</div>
					)
				})}
			</main>
		</div>
	)
}
