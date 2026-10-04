import { useState } from 'preact/hooks'
import { navigateTo } from '../../router'
import type { Milestone } from '../../schemas/models'
import { filters, milestones, tasks } from '../../state/store'
import { t } from '../../utils/i18n'
import { isDoneStatus } from '../../utils/status'

export function MilestoneCards() {
	const milestoneList = milestones.value
	const allTasks = tasks.value
	const currentMilestone = filters.value.selectedMilestone
	const [showCompleted, setShowCompleted] = useState(false)

	if (milestoneList.length === 0) {
		return null
	}

	function isCompleted(m: Milestone): boolean {
		if (m.status === 'closed') return true
		const mTasks = allTasks.filter((t) => t.milestone === m.id)
		return mTasks.length > 0 && mTasks.every((t) => isDoneStatus(t.status))
	}

	const openMilestones = milestoneList.filter((m) => !isCompleted(m))
	const completedMilestones = milestoneList.filter((m) => isCompleted(m))

	const visibleMilestones = showCompleted ? milestoneList : openMilestones

	function handleMilestoneClick(mId: string) {
		if (currentMilestone === mId) {
			navigateTo('board')
		} else {
			navigateTo(`milestone/${mId}`)
		}
	}

	return (
		<section
			class='w-full px-3 sm:px-4 md:px-6 pt-2 pb-0.5'
			aria-label={t('arial_milestones_roadmap')}
		>
			<div class='flex items-center gap-2 overflow-x-auto no-scrollbar py-1'>
				{visibleMilestones.map((m) => {
					const isSelected = currentMilestone === m.id
					const mTasks = allTasks.filter((t) => t.milestone === m.id)
					const doneTasks = mTasks.filter((t) => isDoneStatus(t.status))
					const completed = isCompleted(m)

					return (
						<button
							key={m.id}
							type='button'
							onClick={() => handleMilestoneClick(m.id)}
							aria-pressed={isSelected}
							aria-label={`Filter tasks by milestone ${m.title}`}
							class={`w-56 sm:w-64 p-2.5 rounded-xl border text-left transition-all shrink-0 select-none ${
								isSelected
									? 'border-primary bg-primary/10 ring-1 ring-primary shadow-xs'
									: 'border-base-300/80 bg-base-100 hover:border-primary/40 hover:bg-base-200/40'
							}`}
						>
							<div class='flex items-center justify-between gap-1.5'>
								<h3 class='text-xs sm:text-sm font-semibold text-base-content truncate'>
									{m.title}
								</h3>
								<div class='flex items-center gap-1 shrink-0'>
									{completed && (
										<span
											class='badge badge-xs badge-success text-[10px]'
											title='Milestone ready'
										>
											done
										</span>
									)}
									{mTasks.length > 0 && (
										<span class='text-[10px] font-mono text-base-content/50'>
											{doneTasks.length}/{mTasks.length}
										</span>
									)}
								</div>
							</div>
							{(m.target_timeframe || m.target_date) && (
								<div
									class='flex items-center gap-1 mt-1 text-[10px] text-base-content/60 font-medium'
									data-testid={`milestone-timeframe-${m.id}`}
								>
									<svg
										xmlns='http://www.w3.org/2000/svg'
										class='size-3 text-primary/70 shrink-0'
										fill='none'
										viewBox='0 0 24 24'
										stroke='currentColor'
										aria-hidden='true'
									>
										<path
											stroke-linecap='round'
											stroke-linejoin='round'
											stroke-width='2'
											d='M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z'
										/>
									</svg>
									<span class='truncate'>
										{m.target_timeframe || m.target_date}
									</span>
								</div>
							)}
							{m.summary && (
								<p class='text-[11px] text-base-content/65 line-clamp-2 mt-1 leading-snug'>
									{m.summary}
								</p>
							)}
						</button>
					)
				})}

				{/* Show/Hide Completed Button */}
				{completedMilestones.length > 0 && (
					<button
						type='button'
						onClick={() => setShowCompleted(!showCompleted)}
						class='btn btn-ghost btn-xs text-[11px] rounded-lg text-base-content/60 hover:text-base-content whitespace-nowrap shrink-0'
						aria-expanded={showCompleted}
					>
						{showCompleted
							? t('hide_completed')
							: `${t('show_completed')} (${completedMilestones.length})`}
					</button>
				)}
			</div>
		</section>
	)
}
