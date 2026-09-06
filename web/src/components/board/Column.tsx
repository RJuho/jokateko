import { useEffect, useRef, useState } from 'preact/hooks'
import type { Column as ColumnType, Task } from '../../schemas/models'
import {
	activeColumnDetailId,
	columnSortModes,
	config,
	createTaskInitialColumnId,
	isCreateTaskModalOpen,
	mode,
	type SortMode,
	setColumnSortMode,
} from '../../state/store'
import { t } from '../../utils/i18n'
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
	const [isSortOpen, setIsSortOpen] = useState(false)
	const sortContainerRef = useRef<HTMLDivElement>(null)
	const currentSort = columnSortModes.value[column.id] || 'default'

	const sortOptions: { id: SortMode; label: string }[] = [
		{ id: 'default', label: t('sort_default') },
		{ id: 'priority', label: t('sort_priority') },
		{ id: 'target_at', label: t('sort_target_at') },
		{ id: 'changed_at', label: t('sort_changed_at') },
		{ id: 'created_at', label: t('sort_created_at') },
		{ id: 'title', label: t('sort_title') },
	]

	useEffect(() => {
		if (!isSortOpen) return
		function handleClickOutside(e: MouseEvent) {
			if (
				sortContainerRef.current
				&& !sortContainerRef.current.contains(e.target as Node)
			) {
				setIsSortOpen(false)
			}
		}
		function handleKeyDown(e: KeyboardEvent) {
			if (e.key === 'Escape') {
				setIsSortOpen(false)
			}
		}
		document.addEventListener('click', handleClickOutside)
		document.addEventListener('keydown', handleKeyDown)
		return () => {
			document.removeEventListener('click', handleClickOutside)
			document.removeEventListener('keydown', handleKeyDown)
		}
	}, [isSortOpen])

	const isLive = mode.value === 'live'
	const creatableStates = config.value.board.creatable_states ?? ['backlog']
	const isCreatable =
		creatableStates.length === 0 || creatableStates.includes(column.id)

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
			class={`flex flex-col bg-base-200/40 rounded-2xl border border-base-200 p-3 w-[calc(100vw-2.5rem)] sm:w-104 shrink-0 snap-center transition-all h-full min-h-0 ${
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
				data-testid={`column-header-${column.id}`}
			>
				{/* Left: Button to open Column Details modal */}
				<button
					type='button'
					class='flex items-center gap-2 min-w-0 flex-1 text-left cursor-pointer hover:opacity-85 transition-opacity'
					onClick={() => {
						activeColumnDetailId.value = column.id
					}}
					aria-label={`Column info: ${column.name}`}
					title={`Click to view workflow guidance and prompt for ${column.name}`}
					data-testid={`column-header-pill-${column.id}`}
				>
					<span
						class={`${
							tasks.length > 99
								? 'px-1.5 h-5 rounded-md min-w-5'
								: 'w-5 h-5 rounded-full'
						} flex items-center justify-center text-[11px] font-bold bg-base-100 text-base-content shadow-2xs shrink-0`}
						data-testid='column-task-count'
						title={`${tasks.length} tasks in ${column.name}`}
					>
						{tasks.length}
					</span>
					<h3 class='text-xs sm:text-sm font-bold text-slate-900 tracking-tight truncate'>
						{column.name}
					</h3>
					{column.handled_by && (
						<div
							class='tooltip tooltip-bottom shrink-0'
							data-tip={
								column.instructions
									? `${column.handled_by}: ${column.instructions}`
									: `Handled by ${column.handled_by}`
							}
						>
							<span
								class='badge badge-xs bg-black/15 text-slate-900 border-none font-medium px-1.5 py-0.5'
								data-testid={`column-role-${column.id}`}
							>
								{column.handled_by}
							</span>
						</div>
					)}
				</button>

				{/* Right: Actions (Plus Button + Sort Dropdown) */}
				<div class='flex items-center gap-1 shrink-0 ml-1.5'>
					{/* Plus Button to add a new task to this column */}
					{isLive && isCreatable && (
						<button
							type='button'
							onClick={() => {
								createTaskInitialColumnId.value = column.id
								isCreateTaskModalOpen.value = true
							}}
							class='flex size-5 shrink-0 items-center justify-center rounded-full text-slate-900/80 transition-colors hover:bg-black/15 hover:text-slate-900 cursor-pointer'
							aria-label={`Add new task to ${column.name}`}
							data-testid={`add-task-${column.id}`}
						>
							<svg
								class='size-3.5'
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

					{/* Column Sort Dropdown */}
					<div
						ref={sortContainerRef}
						class={`dropdown dropdown-end ${isSortOpen ? 'dropdown-open' : ''}`}
					>
						<button
							type='button'
							onClick={(e) => {
								e.stopPropagation()
								setIsSortOpen(!isSortOpen)
							}}
							class={`flex size-5 shrink-0 items-center justify-center rounded-full transition-colors cursor-pointer ${
								currentSort !== 'default'
									? 'bg-black/25 text-slate-900 font-bold ring-1 ring-black/30'
									: 'text-slate-900/80 hover:bg-black/15 hover:text-slate-900'
							}`}
							aria-expanded={isSortOpen}
							aria-haspopup='menu'
							aria-label={`${t('arial_sort_tasks')}: ${column.name}`}
							title={`${t('sort_by')}: ${
								sortOptions.find((o) => o.id === currentSort)?.label
								|| t('sort_by')
							}`}
							data-testid={`column-sort-button-${column.id}`}
						>
							<svg
								class='size-3.5'
								fill='none'
								viewBox='0 0 24 24'
								stroke='currentColor'
								stroke-width='2'
								aria-hidden='true'
							>
								<title>Sort column</title>
								<path
									stroke-linecap='round'
									stroke-linejoin='round'
									d='M3 4h13M3 8h9m-9 4h6m4 0l4-4m0 0l4 4m-4-4v12'
								/>
							</svg>
						</button>

						{isSortOpen && (
							<ul
								class='dropdown-content menu menu-xs bg-base-100 text-base-content rounded-box z-50 w-52 sm:w-56 p-1.5 shadow-xl border border-base-200 mt-1.5'
								aria-label={t('arial_sort_options')}
								data-testid={`column-sort-menu-${column.id}`}
							>
								<li class='menu-title px-2 py-1 text-[10px] font-semibold text-base-content/60 uppercase tracking-wider'>
									{t('sort_by')} ({column.name})
								</li>
								{sortOptions.map((opt) => (
									<li key={opt.id}>
										<button
											type='button'
											class={`flex items-center justify-between py-1.5 px-2 rounded-md ${
												currentSort === opt.id ? 'active font-semibold' : ''
											}`}
											onClick={(e) => {
												e.stopPropagation()
												setColumnSortMode(column.id, opt.id)
												setIsSortOpen(false)
											}}
											data-testid={`column-sort-option-${column.id}-${opt.id}`}
										>
											<span>{opt.label}</span>
											{currentSort === opt.id && (
												<svg
													class='size-3.5 text-primary'
													fill='none'
													viewBox='0 0 24 24'
													stroke='currentColor'
													stroke-width='2.5'
													aria-hidden='true'
												>
													<path
														stroke-linecap='round'
														stroke-linejoin='round'
														d='M5 13l4 4L19 7'
													/>
												</svg>
											)}
										</button>
									</li>
								))}
							</ul>
						)}
					</div>
				</div>
			</div>

			{/* Task Cards List: Responsive column height without double body scrollbar */}
			<ul
				class='flex flex-col gap-2.5 overflow-y-auto flex-1 min-h-0 pr-0.5'
				aria-label={`Tasks in ${column.name}`}
			>
				{tasks.map((task) => (
					<li key={task.id} class='list-none'>
						<TaskCard task={task} onDragStart={onDragStart} />
					</li>
				))}

				{tasks.length === 0 && (
					<li class='list-none flex flex-col items-center justify-center py-10 text-center border-2 border-dashed border-base-300/60 rounded-xl text-base-content/40 flex-1 min-h-[120px]'>
						<span class='text-xs font-medium'>{t('no_tasks')}</span>
					</li>
				)}
			</ul>
		</section>
	)
}
