import { ChevronLeft, ChevronRight } from 'lucide-preact'
import { useEffect, useMemo, useRef, useState } from 'preact/hooks'
import { navigateTo, navigateToCalendar } from '../../router'
import type { Milestone, Task } from '../../schemas/models'
import {
	calendarCurrentDate,
	calendarViewMode,
	config,
	createTaskInitialTargetDate,
	filteredTasks,
	filters,
	isCreateTaskModalOpen,
	milestones,
	mode,
} from '../../state/store'
import {
	type CalendarDayInfo,
	type CalendarWeekInfo,
	extractDateKey,
	formatMonthYearTitle,
	formatWeekTitle,
	formatYearMonth,
	formatYearWeek,
	getISOWeek,
	getMonthCalendarWeeks,
	getWeekCalendar,
	monthName,
	weekdayName,
} from '../../utils/date'
import { t, tf, uiLocale } from '../../utils/i18n'
import { isDoneStatus } from '../../utils/status'
import { FilterBar } from '../common/FilterBar'

interface MilestoneSpan {
	milestone: Milestone
	startIndex: number // 0..6
	endIndex: number // 0..6
	isStart: boolean
	isEnd: boolean
}

export function CalendarView() {
	const currentDate = calendarCurrentDate.value
	const viewMode = calendarViewMode.value
	const isWeekView = viewMode === 'week'
	const taskList = filteredTasks.value
	const allMilestones = milestones.value
	const columns = config.value.board.columns

	// Mobile week column horizontal scroll state & ref
	const weekScrollRef = useRef<HTMLDivElement>(null)
	const [activeMobileDay, setActiveMobileDay] = useState<string>('')

	// Weekday and month names come from Intl in the project (or browser) locale
	const locale = uiLocale()
	const weekdayNames = useMemo(
		() => Array.from({ length: 7 }, (_, i) => weekdayName(i, 'short', locale)),
		[locale],
	)

	// Month / Year or Week Title
	const currentYear = currentDate.getFullYear()
	const currentMonth = currentDate.getMonth()

	const periodTitle = useMemo(() => {
		if (isWeekView) {
			const { year: isoYear, week: isoWeek } = getISOWeek(currentDate)
			return formatWeekTitle(isoYear, isoWeek, locale)
		}
		return formatMonthYearTitle(currentYear, currentMonth, locale)
	}, [isWeekView, currentDate, currentYear, currentMonth, locale])

	// Calculate weeks and days based on viewMode
	const weeks: CalendarWeekInfo[] = useMemo(() => {
		if (isWeekView) {
			const { year: isoYear, week: isoWeek } = getISOWeek(currentDate)
			return [getWeekCalendar(isoYear, isoWeek)]
		}
		return getMonthCalendarWeeks(currentYear, currentMonth)
	}, [isWeekView, currentDate, currentYear, currentMonth])

	// Map column id to color
	const columnColorMap = useMemo(() => {
		const map: Record<string, string> = {}
		for (const col of columns) {
			map[col.id] = col.color
		}
		return map
	}, [columns])

	// Index tasks by scheduled date key
	const tasksByDate = useMemo(() => {
		const map: Record<
			string,
			{ task: Task; isTargeted: boolean; isDone: boolean }[]
		> = {}

		for (const task of taskList) {
			const isDone = isDoneStatus(task.status)
			let dateKey: string | null = null
			let isTargeted = false

			if (isDone) {
				// Done tasks: placed on changed_at date
				dateKey = extractDateKey(task.changed_at)
			} else if (task.target_at) {
				// Active tasks with target: placed on target_at
				dateKey = extractDateKey(task.target_at)
				isTargeted = true
			} else {
				// Active tasks without target: placed on changed_at
				dateKey = extractDateKey(task.changed_at)
				isTargeted = false
			}

			if (dateKey) {
				if (!map[dateKey]) {
					map[dateKey] = []
				}
				map[dateKey].push({ task, isTargeted, isDone })
			}
		}

		return map
	}, [taskList])

	// Calculate milestone spans per week row
	const weekMilestoneSpans = useMemo(() => {
		const activeMilestoneId = filters.value.selectedMilestone

		return weeks.map((week) => {
			const weekStartKey = week.days[0].dateKey
			const weekEndKey = week.days[6].dateKey
			const dayKeys = week.days.map((d) => d.dateKey)
			const spans: MilestoneSpan[] = []

			for (const m of allMilestones) {
				if (activeMilestoneId && m.id !== activeMilestoneId) {
					continue
				}

				let startKey = extractDateKey(m.target_start_at || m.target_date)
				let endKey = extractDateKey(
					m.target_end_at || m.target_date || m.target_start_at,
				)

				if (!startKey && !endKey) continue
				if (!startKey) startKey = endKey
				if (!endKey) endKey = startKey
				if (!startKey || !endKey) continue

				if (startKey > endKey) {
					const tmp = startKey
					startKey = endKey
					endKey = tmp
				}

				// Check overlap with this week
				if (startKey <= weekEndKey && endKey >= weekStartKey) {
					let startIndex = 0
					let endIndex = 6

					for (let i = 0; i < 7; i++) {
						if (dayKeys[i] >= startKey) {
							startIndex = i
							break
						}
					}
					for (let i = 6; i >= 0; i--) {
						if (dayKeys[i] <= endKey) {
							endIndex = i
							break
						}
					}

					spans.push({
						milestone: m,
						startIndex,
						endIndex,
						isStart: startKey >= weekStartKey,
						isEnd: endKey <= weekEndKey,
					})
				}
			}

			return spans
		})
	}, [weeks, allMilestones, filters.value.selectedMilestone])

	// Map milestones to dates for mobile week view
	const milestonesByDate = useMemo(() => {
		const map: Record<string, Milestone[]> = {}
		const activeMilestoneId = filters.value.selectedMilestone

		for (const m of allMilestones) {
			if (activeMilestoneId && m.id !== activeMilestoneId) {
				continue
			}

			let startKey = extractDateKey(m.target_start_at || m.target_date)
			let endKey = extractDateKey(
				m.target_end_at || m.target_date || m.target_start_at,
			)

			if (!startKey && !endKey) continue
			if (!startKey) startKey = endKey
			if (!endKey) endKey = startKey
			if (!startKey || !endKey) continue

			if (startKey > endKey) {
				const tmp = startKey
				startKey = endKey
				endKey = tmp
			}

			if (isWeekView && weeks[0]) {
				for (const d of weeks[0].days) {
					if (d.dateKey >= startKey && d.dateKey <= endKey) {
						if (!map[d.dateKey]) map[d.dateKey] = []
						map[d.dateKey].push(m)
					}
				}
			}
		}

		return map
	}, [allMilestones, filters.value.selectedMilestone, isWeekView, weeks])

	// Auto-scroll to current day column when week view opens on mobile
	useEffect(() => {
		if (isWeekView && weeks[0]) {
			const currentKey = extractDateKey(currentDate.toISOString())
			const days = weeks[0].days
			const matchingDay = days.find((d) => d.dateKey === currentKey)
			const targetKey = matchingDay ? matchingDay.dateKey : days[0]?.dateKey
			if (targetKey) {
				setActiveMobileDay(targetKey)
				const el = document.getElementById(`calendar-day-col-${targetKey}`)
				el?.scrollIntoView({
					behavior: 'smooth',
					inline: 'center',
					block: 'nearest',
				})
			}
		}
	}, [isWeekView, currentDate, weeks])

	function scrollToDay(dateKey: string) {
		setActiveMobileDay(dateKey)
		const el = document.getElementById(`calendar-day-col-${dateKey}`)
		el?.scrollIntoView({
			behavior: 'smooth',
			inline: 'center',
			block: 'nearest',
		})
	}

	function handleWeekScroll() {
		if (!isWeekView) return
		const el = weekScrollRef.current
		if (!el) return
		const scrollLeft = el.scrollLeft
		const viewCenter = scrollLeft + el.clientWidth / 2

		const days = weeks[0]?.days || []
		let closestDayKey = days[0]?.dateKey
		let minDistance = Infinity

		for (const day of days) {
			const dayEl = document.getElementById(`calendar-day-col-${day.dateKey}`)
			if (dayEl) {
				const dayCenter = dayEl.offsetLeft + dayEl.offsetWidth / 2
				const distance = Math.abs(viewCenter - dayCenter)
				if (distance < minDistance) {
					minDistance = distance
					closestDayKey = day.dateKey
				}
			}
		}

		if (closestDayKey && closestDayKey !== activeMobileDay) {
			setActiveMobileDay(closestDayKey)
		}
	}

	// Navigation actions
	function handlePrev() {
		const d = new Date(currentDate)
		if (isWeekView) {
			d.setDate(d.getDate() - 7)
			calendarCurrentDate.value = d
			navigateToCalendar({ period: formatYearWeek(d) })
		} else {
			d.setMonth(d.getMonth() - 1)
			calendarCurrentDate.value = d
			navigateToCalendar({ period: formatYearMonth(d) })
		}
	}

	function handleNext() {
		const d = new Date(currentDate)
		if (isWeekView) {
			d.setDate(d.getDate() + 7)
			calendarCurrentDate.value = d
			navigateToCalendar({ period: formatYearWeek(d) })
		} else {
			d.setMonth(d.getMonth() + 1)
			calendarCurrentDate.value = d
			navigateToCalendar({ period: formatYearMonth(d) })
		}
	}

	function handleToday() {
		const today = new Date()
		calendarCurrentDate.value = today
		if (isWeekView) {
			navigateToCalendar({ period: formatYearWeek(today) })
		} else {
			navigateToCalendar({ period: formatYearMonth(today) })
		}
	}

	function handleSwitchView(mode: 'month' | 'week') {
		calendarViewMode.value = mode
		if (mode === 'week') {
			navigateToCalendar({ period: formatYearWeek(currentDate) })
		} else {
			navigateToCalendar({ period: formatYearMonth(currentDate) })
		}
	}

	// Same rule as the board column "+": creating needs the live daemon and a
	// creatable default state. Static exports and client mode have no backend.
	const boardCfg = config.value.board
	const creatableStates = boardCfg.creatable_states ?? ['backlog']
	const canCreateTask =
		mode.value === 'live'
		&& (creatableStates.length === 0
			|| creatableStates.includes(boardCfg.default_create_state || 'backlog'))

	function handleAddTaskForDate(e: MouseEvent, dateKey: string) {
		e.stopPropagation()
		if (!canCreateTask) {
			return
		}
		createTaskInitialTargetDate.value = `${dateKey}T09:00`
		isCreateTaskModalOpen.value = true
	}

	function handleDayNumberClick(e: MouseEvent, day: CalendarDayInfo) {
		e.stopPropagation()
		if (!isWeekView) {
			calendarCurrentDate.value = day.date
			calendarViewMode.value = 'week'
			navigateToCalendar({ period: formatYearWeek(day.date) })
		}
	}

	function handleTaskClick(e: MouseEvent, taskId: string) {
		e.stopPropagation()
		navigateToCalendar({ taskId })
	}

	function handleMilestoneClick(e: MouseEvent, milestoneId: string) {
		e.stopPropagation()
		navigateTo(`milestone/${milestoneId}`)
	}

	return (
		<main
			class='flex-1 min-h-0 flex flex-col overflow-hidden bg-base-100'
			data-testid='calendar-view'
		>
			{/* Calendar Controls & Month/Week Bar */}
			<section
				class='w-full px-3 sm:px-4 md:px-6 py-2.5 flex flex-wrap items-center justify-between gap-2.5 border-b border-base-200/80 bg-base-100/60 backdrop-blur-xs shrink-0'
				aria-label={t('arial_calendar_nav')}
			>
				{/* Left: Navigation Buttons & Period Title */}
				<div class='flex items-center gap-2 sm:gap-3'>
					<div class='join'>
						<button
							type='button'
							onClick={handlePrev}
							class='join-item btn btn-sm btn-ghost btn-square'
							aria-label={t('arial_prev_month')}
							data-testid='calendar-prev-button'
						>
							<ChevronLeft class='size-4' />
						</button>
						<button
							type='button'
							onClick={handleToday}
							class='join-item btn btn-sm btn-ghost text-xs px-2.5 font-medium'
							aria-label={t('arial_today_btn')}
							data-testid='calendar-today-button'
						>
							{t('today')}
						</button>
						<button
							type='button'
							onClick={handleNext}
							class='join-item btn btn-sm btn-ghost btn-square'
							aria-label={t('arial_next_month')}
							data-testid='calendar-next-button'
						>
							<ChevronRight class='size-4' />
						</button>
					</div>

					<h2
						class='text-base sm:text-lg font-bold tracking-tight text-base-content select-none'
						data-testid='calendar-period-title'
					>
						{periodTitle}
					</h2>
				</div>

				{/* Right: Month / Week View Mode Selector */}
				<div class='flex items-center gap-2'>
					<div class='join p-0.5 bg-base-200/60 rounded-xl'>
						<button
							type='button'
							onClick={() => handleSwitchView('month')}
							class={`join-item btn btn-xs sm:btn-sm border-none ${
								viewMode === 'month'
									? 'btn-primary shadow-xs'
									: 'btn-ghost text-base-content/70'
							}`}
							aria-label={t('arial_month_view')}
							data-testid='calendar-view-month'
						>
							{t('month')}
						</button>
						<button
							type='button'
							onClick={() => handleSwitchView('week')}
							class={`join-item btn btn-xs sm:btn-sm border-none ${
								viewMode === 'week'
									? 'btn-primary shadow-xs'
									: 'btn-ghost text-base-content/70'
							}`}
							aria-label={t('arial_week_view')}
							data-testid='calendar-view-week'
						>
							{t('week')}
						</button>
					</div>
				</div>
			</section>

			{/* FilterBar Integration (Syncs search, tags, milestone, and priority) */}
			<div class='shrink-0'>
				<FilterBar />
			</div>

			{/* Calendar Grid Container */}
			<div class='flex-1 min-h-0 flex flex-col p-2 sm:p-3 md:p-4 overflow-y-auto'>
				{/* Mobile Week Days Quick Navigation Bar */}
				{isWeekView && weeks[0] && (
					<nav
						class='flex items-center gap-1.5 px-1 pb-2 shrink-0 overflow-x-auto select-none md:hidden'
						aria-label={t('arial_week_day_nav')}
						data-testid='week-day-quick-nav'
					>
						{weeks[0].days.map((day) => {
							const isSelected = activeMobileDay === day.dateKey
							const count = (tasksByDate[day.dateKey] || []).length
							const dayName =
								weekdayNames[day.dayOfWeekIndex] ?? weekdayNames[0]

							return (
								<button
									key={day.dateKey}
									type='button'
									onClick={() => scrollToDay(day.dateKey)}
									class={`btn btn-xs rounded-lg gap-1 whitespace-nowrap border-none flex-1 ${
										isSelected
											? 'btn-primary btn-soft font-bold shadow-xs'
											: 'btn-ghost font-medium text-base-content/70'
									} ${day.isToday ? 'ring-1 ring-primary/50' : ''}`}
									aria-pressed={isSelected}
									aria-label={tf('arial_calendar_day', {
										day: dayName,
										date: day.date.getDate(),
										count,
									})}
									data-testid={`quick-nav-day-${day.dateKey}`}
								>
									<span>{dayName}</span>
									<span class='font-mono'>{day.date.getDate()}</span>
									{count > 0 && (
										<span class='badge badge-xs badge-neutral'>{count}</span>
									)}
								</button>
							)
						})}
					</nav>
				)}

				<div class='flex-1 min-h-[500px] flex flex-col border border-base-200 rounded-2xl overflow-hidden bg-base-100 shadow-xs'>
					{/* Weekday Column Headers (Mon to Sun + Week column on left) */}
					<div
						class={`${
							isWeekView ? 'hidden md:grid' : 'grid'
						} grid-cols-[38px_repeat(7,1fr)] sm:grid-cols-[46px_repeat(7,1fr)] border-b border-base-200 bg-base-200/50 text-xs font-semibold text-base-content/70 select-none`}
					>
						<div
							class='p-2 text-center text-base-content/40 font-mono border-r border-base-200/60'
							title={t('week_num', 'Week')}
						>
							#
						</div>
						{weekdayNames.map((dayName, idx) => (
							<div
								key={idx}
								class={`p-2 text-center font-medium ${
									idx === 5 || idx === 6
										? 'text-base-content/50'
										: 'text-base-content/80'
								} ${idx < 6 ? 'border-r border-base-200/60' : ''}`}
							>
								{dayName}
							</div>
						))}
					</div>

					{/* Calendar Week Rows */}
					<div class='flex-1 flex flex-col divide-y divide-base-200/70 min-h-0'>
						{weeks.map((week, weekIdx) => {
							const milestoneSpans = weekMilestoneSpans[weekIdx] || []

							return (
								<div
									key={`week-${week.weekNumber}-${week.weekYear}`}
									class={`flex-1 shrink-0 ${
										isWeekView
											? 'flex flex-col min-h-0 md:min-h-[420px] md:grid md:grid-cols-[46px_repeat(7,1fr)]'
											: `${
													milestoneSpans.length > 0
														? 'min-h-[145px] sm:min-h-[160px]'
														: 'min-h-[95px] sm:min-h-[115px]'
												} grid grid-cols-[38px_repeat(7,1fr)] sm:grid-cols-[46px_repeat(7,1fr)]`
									} relative`}
									data-testid={`calendar-week-row-${week.weekNumber}`}
								>
									{/* Left Edge: Week Number Indicator */}
									<div
										class={`p-1.5 items-center justify-start border-r border-base-200/60 bg-base-200/25 select-none ${
											isWeekView ? 'hidden md:flex flex-col' : 'flex flex-col'
										}`}
										data-testid='calendar-week-number'
										title={tf('arial_week_title', { week: week.weekNumber })}
									>
										<span class='text-[10px] sm:text-xs font-mono font-medium text-base-content/45 mt-1'>
											{week.weekNumber}
										</span>
									</div>

									{/* 7 Days in Week Row */}
									<div
										ref={isWeekView ? weekScrollRef : undefined}
										onScroll={isWeekView ? handleWeekScroll : undefined}
										data-testid={
											isWeekView ? 'calendar-week-scroll-container' : undefined
										}
										class={
											isWeekView
												? 'flex-1 flex flex-row overflow-x-auto snap-x snap-mandatory scroll-smooth gap-3 p-2.5 sm:p-3 min-h-0 items-stretch md:col-span-7 md:grid md:grid-cols-7 md:relative md:divide-x md:divide-base-200/60 md:p-0 md:gap-0 md:snap-none md:overflow-visible'
												: 'col-span-7 grid grid-cols-7 relative divide-x divide-base-200/60'
										}
									>
										{/* Desktop Milestone Timeline Track Overlay (above day cells) */}
										{milestoneSpans.length > 0 && (
											<div
												class={`absolute inset-x-0 top-7 z-10 px-1 pointer-events-none flex flex-col gap-1 ${
													isWeekView ? 'hidden md:flex' : ''
												}`}
											>
												{milestoneSpans.map((span) => {
													const colStart = span.startIndex + 1
													const colSpan = span.endIndex - span.startIndex + 1
													const percent = Math.round(
														span.milestone.progress_percentage ?? 0,
													)

													return (
														<button
															key={span.milestone.id}
															type='button'
															onClick={(e) =>
																handleMilestoneClick(e, span.milestone.id)
															}
															class={`pointer-events-auto h-5 sm:h-5.5 text-left text-[10px] sm:text-xs font-medium flex items-center justify-between px-2 text-white shadow-xs transition-transform hover:scale-[1.008] cursor-pointer group truncate ${
																span.isStart ? 'rounded-l-lg' : 'rounded-l-none'
															} ${span.isEnd ? 'rounded-r-lg' : 'rounded-r-none'}`}
															style={{
																gridColumnStart: colStart,
																gridColumnEnd: `span ${colSpan}`,
																marginLeft: `${(span.startIndex / 7) * 100}%`,
																width: `${(colSpan / 7) * 100}%`,
																backgroundColor:
																	span.milestone.status === 'closed'
																		? '#64748b'
																		: '#6366f1',
															}}
															data-testid={`calendar-milestone-${span.milestone.id}`}
															title={tf('arial_milestone_progress', {
																title: span.milestone.title,
																percent,
															})}
														>
															<span class='truncate font-semibold'>
																{span.milestone.title}
															</span>
															<span class='text-[9px] opacity-90 shrink-0 font-mono ml-1.5'>
																{percent}%
															</span>
														</button>
													)
												})}
											</div>
										)}

										{/* Day Cells (Monday to Sunday) */}
										{week.days.map((day: CalendarDayInfo) => {
											const scheduled = tasksByDate[day.dateKey] || []
											const hasOverflow = !isWeekView && scheduled.length > 3
											const visibleTasks = hasOverflow
												? scheduled.slice(0, 3)
												: scheduled
											const overflowCount = scheduled.length - 3
											const dayMs = milestonesByDate[day.dateKey] || []

											return (
												<div
													key={day.dateKey}
													id={`calendar-day-col-${day.dateKey}`}
													class={
														isWeekView
															? `w-[calc(100vw-2.5rem)] sm:w-96 shrink-0 snap-center rounded-2xl border border-base-200 bg-base-200/40 p-3 sm:p-3.5 flex flex-col h-full min-h-0 shadow-xs transition-all md:w-auto md:shrink md:snap-none md:rounded-none md:border-0 md:shadow-none md:p-2 md:bg-base-100 md:hover:bg-base-200/30 ${
																	day.isToday
																		? 'ring-2 ring-primary/40 bg-primary/5 md:ring-1 md:ring-inset'
																		: ''
																}`
															: `group relative p-1.5 sm:p-2 flex flex-col transition-colors min-h-0 overflow-hidden ${
																	day.isCurrentMonth
																		? 'bg-base-100'
																		: 'bg-base-200/20 opacity-40'
																} ${
																	day.isToday
																		? 'bg-primary/5 ring-1 ring-inset ring-primary/40'
																		: 'hover:bg-base-200/30'
																}`
													}
													data-testid={`calendar-day-${day.dateKey}`}
												>
													{/* Day Cell Header */}
													<div
														class={`flex items-center justify-between select-none shrink-0 ${
															isWeekView
																? 'mb-2 pb-1.5 border-b border-base-200/60 md:border-b-0 md:mb-1 md:pb-0'
																: 'mb-1'
														}`}
													>
														<div class='flex items-center gap-1.5'>
															{isWeekView && (
																<span class='font-bold text-sm text-base-content md:hidden'>
																	{weekdayNames[day.dayOfWeekIndex]
																		?? weekdayNames[0]}
																	, {day.date.getDate()}{' '}
																	{monthName(
																		day.date.getMonth(),
																		'short',
																		locale,
																	)}
																</span>
															)}
															<button
																type='button'
																onClick={(e) => handleDayNumberClick(e, day)}
																class={`text-xs sm:text-sm font-semibold inline-flex items-center justify-center size-5 sm:size-6 rounded-full cursor-pointer hover:ring-2 hover:ring-primary/50 transition-all ${
																	isWeekView ? 'hidden md:inline-flex' : ''
																} ${
																	day.isToday
																		? 'bg-primary text-primary-content font-bold'
																		: 'text-base-content/80 hover:bg-base-200'
																}`}
																title={t('view_week_tasks')}
																aria-label={tf('arial_view_week_tasks_date', {
																	date: day.dateKey,
																})}
															>
																{day.dayOfMonth}
															</button>
															{isWeekView && day.isToday && (
																<span class='badge badge-primary badge-xs md:hidden font-semibold'>
																	{t('today')}
																</span>
															)}
														</div>

														<div class='flex items-center gap-1'>
															{isWeekView && scheduled.length > 0 && (
																<span class='badge badge-neutral badge-xs font-mono md:hidden'>
																	{scheduled.length}
																</span>
															)}
															{canCreateTask && (
																<button
																	type='button'
																	onClick={(e) =>
																		handleAddTaskForDate(e, day.dateKey)
																	}
																	class={`btn btn-ghost btn-xs btn-circle size-6 sm:size-5 text-base-content/60 hover:text-primary transition-opacity cursor-pointer ${
																		isWeekView
																			? 'opacity-100 md:opacity-0 md:group-hover:opacity-100'
																			: 'opacity-0 group-hover:opacity-100'
																	}`}
																	title={t('add_task_date')}
																	aria-label={tf('arial_add_task_date', {
																		date: day.dateKey,
																	})}
																	data-testid={`calendar-add-task-${day.dateKey}`}
																>
																	+
																</button>
															)}
														</div>
													</div>

													{/* Mobile In-Column Milestone Pills (Week View) */}
													{isWeekView && dayMs.length > 0 && (
														<div class='flex flex-col gap-1 mb-2 md:hidden shrink-0'>
															{dayMs.map((m) => {
																const percent = Math.round(
																	m.progress_percentage ?? 0,
																)
																return (
																	<button
																		key={m.id}
																		type='button'
																		onClick={(e) =>
																			handleMilestoneClick(e, m.id)
																		}
																		class='h-6 px-2.5 text-xs font-semibold rounded-lg flex items-center justify-between text-white shadow-xs cursor-pointer truncate'
																		style={{
																			backgroundColor:
																				m.status === 'closed'
																					? '#64748b'
																					: '#6366f1',
																		}}
																		title={tf('arial_milestone_progress', {
																			title: m.title,
																			percent,
																		})}
																	>
																		<span class='truncate'>{m.title}</span>
																		<span class='text-[10px] opacity-90 font-mono ml-2 shrink-0'>
																			{percent}%
																		</span>
																	</button>
																)
															})}
														</div>
													)}

													{/* Spacer for desktop milestone bar overlay */}
													{milestoneSpans.length > 0 && (
														<div
															class={`shrink-0 ${
																isWeekView ? 'hidden md:block' : ''
															}`}
															style={{
																height: `${milestoneSpans.length * 24}px`,
															}}
														/>
													)}

													{/* Task Chips Container */}
													<div
														class={`flex-1 flex flex-col ${
															isWeekView
																? 'gap-1.5 sm:gap-2 overflow-y-auto min-h-0 pr-0.5'
																: 'gap-1 overflow-hidden min-h-0'
														}`}
													>
														{visibleTasks.map(
															({ task, isTargeted, isDone }) => {
																const colColor =
																	columnColorMap[task.status] || '#94a3b8'

																return (
																	<button
																		key={task.id}
																		type='button'
																		onClick={(e) => handleTaskClick(e, task.id)}
																		class={`w-full text-left transition-all group/task cursor-pointer flex items-center ${
																			isWeekView
																				? 'rounded-lg px-2.5 py-2 sm:py-2.5 text-xs sm:text-sm font-medium gap-2 shadow-2xs'
																				: 'rounded-md px-1.5 py-0.5 text-[11px] font-medium gap-1.5 truncate'
																		} ${
																			isDone
																				? 'line-through text-base-content/40 bg-base-200/50'
																				: isTargeted
																					? 'bg-base-200/90 hover:bg-base-300/80 text-base-content opacity-100'
																					: 'bg-base-200/60 hover:bg-base-300/70 text-base-content/85 opacity-75'
																		}`}
																		data-testid={`calendar-task-${task.id}`}
																		title={tf('arial_task_status_title', {
																			title: task.title,
																			status: task.status,
																		})}
																	>
																		<span
																			class={`rounded-full shrink-0 ${
																				isWeekView
																					? 'size-2 sm:size-2.5'
																					: 'size-1.5'
																			}`}
																			style={{ backgroundColor: colColor }}
																		/>
																		<span
																			class={`leading-tight ${
																				isWeekView
																					? 'break-words line-clamp-2 sm:line-clamp-3 text-left'
																					: 'truncate'
																			}`}
																		>
																			{task.title}
																		</span>
																	</button>
																)
															},
														)}

														{/* Empty State on Mobile Week View */}
														{isWeekView && scheduled.length === 0 && (
															<div class='flex-1 flex flex-col items-center justify-center p-6 text-center text-base-content/40 md:hidden'>
																<span class='text-xs font-medium'>
																	{t('no_tasks_for_day')}
																</span>
															</div>
														)}

														{/* "+N more" pill: navigates directly to weekly view */}
														{hasOverflow && (
															<div class='mt-auto pt-0.5'>
																<button
																	type='button'
																	onClick={(e) => {
																		e.stopPropagation()
																		calendarCurrentDate.value = day.date
																		calendarViewMode.value = 'week'
																		navigateToCalendar({
																			period: formatYearWeek(day.date),
																		})
																	}}
																	class='btn btn-ghost btn-xs text-[10px] text-primary h-5 min-h-5 px-1 font-semibold hover:bg-primary/10 w-full justify-start cursor-pointer'
																	title={t('view_week_tasks')}
																	aria-label={tf('arial_more_tasks', {
																		count: overflowCount,
																	})}
																	data-testid={`calendar-more-tasks-${day.dateKey}`}
																>
																	+{overflowCount} {t('more')}
																</button>
															</div>
														)}
													</div>
												</div>
											)
										})}
									</div>
								</div>
							)
						})}
					</div>
				</div>
			</div>
		</main>
	)
}
