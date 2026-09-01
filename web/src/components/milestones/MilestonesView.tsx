import { useState } from 'preact/hooks'
import { activeTab, milestones, setMilestoneFilter } from '../../state/store'
import { TagBadge } from '../common/Badge'

export function MilestonesView() {
	const allMilestones = milestones.value
	const [showArchived, setShowArchived] = useState(false)

	const visibleMilestones = allMilestones.filter((m) =>
		showArchived ? true : !m.is_archived,
	)

	function handleFilterBoard(milestoneId: string) {
		setMilestoneFilter(milestoneId)
		activeTab.value = 'board'
	}

	return (
		<div class='flex-1 max-w-7xl w-full mx-auto p-4 sm:p-6 flex flex-col gap-5'>
			{/* Top Bar: Title, Count, Archive Toggle */}
			<div class='flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-base-200 pb-4'>
				<div>
					<h2 class='text-xl sm:text-2xl font-bold tracking-tight text-base-content'>
						Milestones
					</h2>
					<p class='text-xs sm:text-sm text-base-content/60 mt-0.5'>
						Track deliverables, target dates, and roadmap progress
					</p>
				</div>

				<div class='flex items-center gap-2 self-start sm:self-auto'>
					<label class='label cursor-pointer gap-2 py-1'>
						<span class='label-text text-xs font-medium'>Show Archived</span>
						<input
							type='checkbox'
							checked={showArchived}
							onChange={(e) =>
								setShowArchived((e.target as HTMLInputElement).checked)
							}
							class='toggle toggle-primary toggle-sm'
							aria-label='Toggle archived milestones'
						/>
					</label>
				</div>
			</div>

			{/* Milestones Grid */}
			<div class='grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4'>
				{visibleMilestones.map((m) => {
					const isClosed = m.status === 'closed'
					const percent = Math.round(m.progress_percentage || 0)

					return (
						<div
							key={m.id}
							class='card bg-base-100 border border-base-200/80 shadow-xs hover:shadow-md transition-all rounded-2xl p-5 flex flex-col justify-between gap-4'
						>
							{/* Card Header */}
							<div class='flex flex-col gap-2'>
								<div class='flex items-center justify-between gap-2'>
									<span
										class={`badge badge-sm font-semibold uppercase tracking-wider text-[10px] ${
											isClosed
												? 'badge-neutral text-neutral-content'
												: 'badge-primary text-primary-content'
										}`}
									>
										{m.status}
									</span>

									{m.target_date && (
										<div class='flex items-center gap-1 text-[11px] font-mono text-base-content/60'>
											<svg
												xmlns='http://www.w3.org/2000/svg'
												class='h-3.5 w-3.5'
												fill='none'
												viewBox='0 0 24 24'
												stroke='currentColor'
												aria-hidden='true'
											>
												<title>Target date</title>
												<path
													stroke-linecap='round'
													stroke-linejoin='round'
													stroke-width='2'
													d='M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z'
												/>
											</svg>
											<span>{m.target_date}</span>
										</div>
									)}
								</div>

								<h3 class='text-base font-bold text-base-content leading-snug'>
									{m.title}
								</h3>

								{m.summary && (
									<p class='text-xs text-base-content/70 leading-relaxed line-clamp-3'>
										{m.summary}
									</p>
								)}
							</div>

							{/* Progress Bar & Counters */}
							<div class='flex flex-col gap-1.5 pt-2 border-t border-base-200/60'>
								<div class='flex items-center justify-between text-xs font-semibold'>
									<span class='text-base-content/70'>Progress</span>
									<span class='text-primary'>{percent}%</span>
								</div>
								<progress
									class='progress progress-primary w-full h-2 rounded-full'
									value={percent}
									max='100'
									aria-label={`${m.title} progress: ${percent}%`}
								/>
								<div class='flex items-center justify-between text-[11px] text-base-content/50 pt-0.5'>
									<span>
										{m.completed_tasks || 0} of {m.total_tasks || 0} tasks done
									</span>
									<button
										type='button'
										onClick={() => handleFilterBoard(m.id)}
										class='text-primary hover:underline font-medium'
										aria-label={`View tasks for milestone ${m.title}`}
									>
										View Tasks →
									</button>
								</div>
							</div>

							{/* Tags */}
							{m.tags && m.tags.length > 0 && (
								<div class='flex flex-wrap items-center gap-1 pt-1'>
									{m.tags.map((tag) => (
										<TagBadge key={tag} tag={tag} />
									))}
								</div>
							)}
						</div>
					)
				})}

				{visibleMilestones.length === 0 && (
					<div class='col-span-full flex flex-col items-center justify-center py-16 text-center border-2 border-dashed border-base-300 rounded-2xl text-base-content/40'>
						<span class='text-sm font-medium'>No milestones found</span>
					</div>
				)}
			</div>
		</div>
	)
}
