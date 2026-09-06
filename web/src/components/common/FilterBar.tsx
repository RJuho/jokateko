import {
	allTags,
	columnSortModes,
	config,
	configuredPriorities,
	filteredTasks,
	filters,
	resetFilters,
	tasks,
	togglePriorityFilter,
	toggleStateFilter,
	toggleTagFilter,
} from '../../state/store'
import { getContrastTextColor } from '../../utils/colors'
import { t } from '../../utils/i18n'
import { TagBadge } from './Badge'

export function FilterBar() {
	const currentFilters = filters.value
	const tagList = allTags.value
	const priorities = configuredPriorities.value
	const columns = config.value.board.columns
	const hasColumnSorts = Object.values(columnSortModes.value).some(
		(m) => m !== 'default',
	)

	const hasActiveFilters =
		currentFilters.searchQuery !== ''
		|| currentFilters.selectedTags.length > 0
		|| currentFilters.selectedMilestone !== null
		|| currentFilters.selectedPriorities.length > 0
		|| currentFilters.selectedStates.length > 0
		|| hasColumnSorts

	return (
		<aside
			class='bg-base-100/80 backdrop-blur-xs border-b border-base-200 py-1.5 px-3 sm:px-4 md:px-6 sticky top-12 md:top-13 z-20 shadow-2xs'
			aria-label={t('arial_filter_tasks')}
		>
			<div class='w-full flex items-center justify-between gap-3 overflow-x-auto no-scrollbar'>
				{/* Combined State + Priority + Tags */}
				<div class='flex items-center gap-2 shrink-0'>
					{/* State / Column Status Filter Buttons */}
					{columns.length > 0 && (
						<fieldset
							class='flex items-center gap-0.5 border border-base-300 rounded-lg p-0.5 bg-base-200/40 shrink-0'
							aria-label={t('arial_filter_by_state')}
						>
							{columns.map((col) => {
								const isSelected = currentFilters.selectedStates.includes(
									col.id,
								)
								const color = col.color
								const activeStyle =
									isSelected && color
										? {
												backgroundColor: color,
												color: getContrastTextColor(color),
											}
										: undefined

								return (
									<button
										key={col.id}
										type='button'
										onClick={() => toggleStateFilter(col.id)}
										class={`px-2 py-0.5 text-[10px] font-semibold rounded-md transition-all flex items-center gap-1 cursor-pointer ${
											isSelected
												? color
													? 'shadow-xs'
													: 'bg-primary text-primary-content shadow-xs'
												: 'text-base-content/60 hover:text-base-content hover:bg-base-200/60'
										}`}
										style={activeStyle}
										aria-pressed={isSelected}
										aria-label={`Filter state ${col.name}`}
										data-testid={`filter-state-${col.id}`}
									>
										{color && (
											<span
												class='size-1.5 rounded-full shrink-0'
												style={{
													backgroundColor: isSelected
														? getContrastTextColor(color)
														: color,
												}}
											/>
										)}
										<span>{col.name}</span>
									</button>
								)
							})}
						</fieldset>
					)}

					{/* Priority Buttons */}
					<fieldset
						class='flex items-center gap-0.5 border border-base-300 rounded-lg p-0.5 bg-base-200/40 shrink-0'
						aria-label={t('arial_filter_by_priority')}
					>
						{priorities.map((p) => {
							const isSelected = currentFilters.selectedPriorities.includes(
								p.id,
							)
							const color = p.color
							const activeStyle =
								isSelected && color
									? {
											backgroundColor: color,
											color: getContrastTextColor(color),
										}
									: undefined

							return (
								<button
									key={p.id}
									type='button'
									onClick={() => togglePriorityFilter(p.id)}
									class={`px-2 py-0.5 text-[10px] uppercase font-semibold rounded-md transition-all ${
										isSelected
											? color
												? 'shadow-xs'
												: 'bg-primary text-primary-content shadow-xs'
											: 'text-base-content/60 hover:text-base-content'
									}`}
									style={activeStyle}
									aria-pressed={isSelected}
									aria-label={`Filter priority ${p.name}`}
								>
									{p.name.slice(0, 3)}
								</button>
							)
						})}
					</fieldset>

					{/* Tag Badges: No "Tags:" prefix, no "#" */}
					{tagList.length > 0 && (
						<fieldset
							class='flex items-center gap-1.5 overflow-x-auto no-scrollbar py-0.5 shrink-0'
							aria-label={t('arial_filter_by_tags')}
						>
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
				</div>

				{/* Right: Reset Button (if active) + End of row "x / y tasks" */}
				<div class='flex items-center gap-2 shrink-0 ml-auto'>
					{hasActiveFilters && (
						<button
							type='button'
							onClick={resetFilters}
							class='btn btn-ghost btn-xs text-error font-medium text-[11px]'
							aria-label={t('arial_reset_all_filters')}
						>
							{t('reset')}
						</button>
					)}
					<div class='text-[11px] text-base-content/50 font-mono whitespace-nowrap'>
						{filteredTasks.value.length} / {tasks.value.length} {t('tasks')}
					</div>
				</div>
			</div>
		</aside>
	)
}
