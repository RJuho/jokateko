import { useEffect, useRef, useState } from 'preact/hooks'
import type { Task } from '../../schemas/models'
import { columnTasks, config, mode, tasks, upsertTask } from '../../state/store'
import { t, tf } from '../../utils/i18n'
import { Column } from './Column'

export function KanbanBoard() {
	const cols = config.value.board.columns
	const groupedTasks = columnTasks.value
	const isLive = mode.value === 'live'
	const [activeMobileColumn, setActiveMobileColumn] = useState<string>(
		cols[0]?.id || 'backlog',
	)
	const mainRef = useRef<HTMLElement>(null)
	const [needsColumnNav, setNeedsColumnNav] = useState(false)

	// Detect if window does not fit all columns in one screen (scrollWidth > clientWidth)
	useEffect(() => {
		const el = mainRef.current
		if (!el) return

		function checkOverflow() {
			if (!el) return
			const overflows = el.scrollWidth > el.clientWidth + 4
			setNeedsColumnNav(overflows)
		}

		checkOverflow()

		const ro =
			typeof ResizeObserver !== 'undefined'
				? new ResizeObserver(() => checkOverflow())
				: null
		ro?.observe(el)

		window.addEventListener('resize', checkOverflow)

		return () => {
			ro?.disconnect()
			window.removeEventListener('resize', checkOverflow)
		}
	}, [cols.length])

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

	function handleBoardScroll() {
		const el = mainRef.current
		if (!el) return
		const scrollLeft = el.scrollLeft
		const viewCenter = scrollLeft + el.clientWidth / 2

		let closestColId = cols[0]?.id || 'backlog'
		let minDistance = Infinity

		for (const c of cols) {
			const colEl = document.getElementById(`kanban-col-${c.id}`)
			if (colEl) {
				const colCenter = colEl.offsetLeft + colEl.offsetWidth / 2
				const distance = Math.abs(viewCenter - colCenter)
				if (distance < minDistance) {
					minDistance = distance
					closestColId = c.id
				}
			}
		}

		if (closestColId !== activeMobileColumn) {
			setActiveMobileColumn(closestColId)
		}
	}

	return (
		<div class='flex flex-col flex-1 min-h-0 w-full overflow-hidden'>
			{/* Column Quick Navigation: appears automatically when columns do not fit on one screen */}
			{needsColumnNav && (
				<nav
					class='flex flex-wrap items-center gap-1.5 px-3 sm:px-4 md:px-6 py-1.5 sm:py-2 border-b border-base-200/80 shrink-0 bg-base-100/60 transition-all select-none'
					aria-label={t('arial_column_quick_nav')}
					data-testid='column-quick-nav'
				>
					{cols.map((c) => {
						const isSelected = activeMobileColumn === c.id
						const count = groupedTasks[c.id]?.length || 0
						return (
							<button
								key={c.id}
								type='button'
								onClick={() => scrollToColumn(c.id)}
								class={`btn btn-xs rounded-lg gap-1.5 whitespace-nowrap border-none ${
									isSelected ? 'btn-primary btn-soft' : 'btn-ghost'
								}`}
								aria-pressed={isSelected}
								aria-label={tf('arial_scroll_to_column', {
									name: c.name,
									count,
								})}
								data-testid={`quick-nav-col-${c.id}`}
							>
								<span
									class='size-2 rounded-full'
									style={{ backgroundColor: c.color }}
									aria-hidden='true'
								/>
								{c.name}
								<span class='badge badge-xs badge-neutral'>{count}</span>
							</button>
						)
					})}
				</nav>
			)}

			{/* Board Columns: scroll-snap-type x mandatory for single-column mobile view & fixed desktop widths */}
			<main
				ref={mainRef}
				onScroll={handleBoardScroll}
				class='flex-1 overflow-x-auto snap-x snap-mandatory scroll-smooth flex gap-3 sm:gap-4 p-3 sm:p-4 md:p-6 min-h-0 items-stretch'
				aria-label={t('arial_kanban_columns')}
			>
				{cols.map((col) => (
					<div key={col.id} class='shrink-0 snap-center h-full flex flex-col'>
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
