import type { Priority } from '../../schemas/models'
import {
	allTags,
	filteredTasks,
	filters,
	milestones,
	resetFilters,
	setMilestoneFilter,
	setSearchQuery,
	tasks,
	togglePriorityFilter,
	toggleTagFilter,
} from '../../state/store'
import { TagBadge } from './Badge'

export function FilterBar() {
	const currentFilters = filters.value
	const tagList = allTags.value
	const milestoneList = milestones.value
	const priorities: Priority[] = ['critical', 'high', 'medium', 'low']

	const hasActiveFilters =
		currentFilters.searchQuery !== ''
		|| currentFilters.selectedTags.length > 0
		|| currentFilters.selectedMilestone !== null
		|| currentFilters.selectedPriorities.length > 0

	return (
		<search
			class='bg-base-100/80 backdrop-blur-xs border-b border-base-200 py-2.5 px-4 sm:px-6 sticky top-24 md:top-28 z-20'
			aria-label='Filter and search tasks'
		>
			<div class='max-w-7xl mx-auto flex flex-col gap-2.5'>
				{/* Top Row: Search Input + Filter Dropdowns */}
				<div class='flex flex-col sm:flex-row items-stretch sm:items-center gap-2'>
					{/* Search Input */}
					<div class='relative flex-1 min-w-0'>
						<div
							class='absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-base-content/40'
							aria-hidden='true'
						>
							<svg
								xmlns='http://www.w3.org/2000/svg'
								class='h-4 w-4'
								fill='none'
								viewBox='0 0 24 24'
								stroke='currentColor'
								aria-hidden='true'
							>
								<title>Search icon</title>
								<path
									stroke-linecap='round'
									stroke-linejoin='round'
									stroke-width='2'
									d='M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z'
								/>
							</svg>
						</div>
						<input
							type='search'
							placeholder='Search tasks by title, ID, or summary...'
							value={currentFilters.searchQuery}
							onInput={(e) =>
								setSearchQuery((e.target as HTMLInputElement).value)
							}
							class='input input-sm input-bordered w-full pl-9 pr-8 text-xs md:text-sm rounded-lg'
							aria-label='Search tasks by title, ID, or summary'
							data-testid='search-input'
						/>
						{currentFilters.searchQuery && (
							<button
								type='button'
								onClick={() => setSearchQuery('')}
								class='absolute inset-y-0 right-0 pr-2.5 flex items-center text-base-content/40 hover:text-base-content'
								aria-label='Clear search input'
							>
								✕
							</button>
						)}
					</div>

					{/* Dropdowns & Controls */}
					<div class='flex items-center gap-2 overflow-x-auto no-scrollbar shrink-0'>
						{/* Milestone Dropdown */}
						{milestoneList.length > 0 && (
							<select
								value={currentFilters.selectedMilestone || ''}
								onChange={(e) => {
									const val = (e.target as HTMLSelectElement).value
									setMilestoneFilter(val === '' ? null : val)
								}}
								class='select select-sm select-bordered text-xs rounded-lg min-w-[130px]'
								aria-label='Filter tasks by milestone'
								data-testid='milestone-filter'
							>
								<option value=''>All Milestones</option>
								{milestoneList.map((m) => (
									<option key={m.id} value={m.id}>
										{m.title}
									</option>
								))}
							</select>
						)}

						{/* Priority Buttons */}
						<fieldset
							class='flex items-center gap-1 border border-base-300 rounded-lg p-0.5 bg-base-200/50'
							aria-label='Filter by priority'
						>
							{priorities.map((p) => {
								const isSelected = currentFilters.selectedPriorities.includes(p)
								return (
									<button
										key={p}
										type='button'
										onClick={() => togglePriorityFilter(p)}
										class={`px-2 py-0.5 text-[10px] uppercase font-semibold rounded-md transition-all ${
											isSelected
												? 'bg-primary text-primary-content shadow-xs'
												: 'text-base-content/60 hover:text-base-content'
										}`}
										aria-pressed={isSelected}
										aria-label={`Filter priority ${p}`}
									>
										{p.slice(0, 3)}
									</button>
								)
							})}
						</fieldset>

						{/* Reset Button */}
						{hasActiveFilters && (
							<button
								type='button'
								onClick={resetFilters}
								class='btn btn-ghost btn-xs text-error font-medium'
								aria-label='Reset all filters'
							>
								Reset
							</button>
						)}
					</div>
				</div>

				{/* Bottom Row: Tag Chips (Scrollable) */}
				{tagList.length > 0 && (
					<fieldset
						class='flex items-center gap-1.5 overflow-x-auto no-scrollbar py-0.5'
						aria-label='Filter by tags'
					>
						<span class='text-[11px] text-base-content/50 uppercase font-bold tracking-wider shrink-0 mr-1'>
							Tags:
						</span>
						{tagList.map((tag) => (
							<TagBadge
								key={tag}
								tag={tag}
								selected={currentFilters.selectedTags.includes(tag)}
								onClick={() => toggleTagFilter(tag)}
							/>
						))}
					</fieldset>
				)}

				{/* Result Count Info */}
				<div class='text-[11px] text-base-content/60 flex items-center justify-between'>
					<span>
						Showing {filteredTasks.value.length} of {tasks.value.length} tasks
					</span>
				</div>
			</div>
		</search>
	)
}
