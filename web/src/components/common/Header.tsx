import {
	activeTab,
	config,
	connectionStatus,
	isCreateTaskModalOpen,
	mode,
	type Tab,
} from '../../state/store'
import { ModeBadge } from './Badge'

export function Header() {
	const currentTab = activeTab.value
	const isLive = mode.value === 'live'
	const project = config.value.project

	const tabs: Array<{ id: Tab; label: string; testId: string }> = [
		{ id: 'board', label: 'Board', testId: 'tab-board' },
		{ id: 'milestones', label: 'Milestones', testId: 'tab-milestones' },
		{ id: 'strategies', label: 'Strategies', testId: 'tab-strategies' },
		{ id: 'glossary', label: 'Glossary', testId: 'tab-glossary' },
	]

	return (
		<header class='bg-base-100 border-b border-base-300 sticky top-0 z-30 shadow-xs'>
			<div class='max-w-7xl mx-auto px-4 sm:px-6'>
				{/* Top Bar: Title, Mode, New Task Button */}
				<div class='flex items-center justify-between h-14 md:h-16 gap-3'>
					{/* Brand & Title */}
					<div class='flex items-center gap-2.5 min-w-0'>
						<div
							class='w-7 h-7 md:w-8 md:h-8 rounded-md bg-primary text-primary-content flex items-center justify-center font-black text-sm shrink-0 shadow-xs'
							aria-hidden='true'
						>
							J
						</div>
						<div class='min-w-0'>
							<div class='flex items-center gap-2'>
								<h1 class='text-base md:text-lg font-bold tracking-tight truncate'>
									{project.name || 'Jokateko'}
								</h1>
								<ModeBadge
									mode={mode.value}
									connectionStatus={connectionStatus.value}
								/>
							</div>
							{project.description && (
								<p class='text-xs text-base-content/60 truncate hidden sm:block'>
									{project.description}
								</p>
							)}
						</div>
					</div>

					{/* Action Buttons */}
					<div class='flex items-center gap-2 shrink-0'>
						{isLive ? (
							<button
								type='button'
								onClick={() => {
									isCreateTaskModalOpen.value = true
								}}
								class='btn btn-primary btn-sm gap-1.5 font-medium shadow-xs'
								aria-label='Create new task'
								data-testid='create-task-button'
							>
								<svg
									xmlns='http://www.w3.org/2000/svg'
									class='h-4 w-4'
									fill='none'
									viewBox='0 0 24 24'
									stroke='currentColor'
									aria-hidden='true'
								>
									<path
										stroke-linecap='round'
										stroke-linejoin='round'
										stroke-width='2'
										d='M12 4v16m8-8H4'
									/>
								</svg>
								<span class='hidden sm:inline'>New Task</span>
							</button>
						) : (
							<span
								class='badge badge-neutral badge-sm text-xs hidden sm:inline-flex'
								title='Board is viewing a static offline snapshot'
							>
								Read-Only Snapshot
							</span>
						)}
					</div>
				</div>

				{/* Bottom Bar: Tab Navigation (Mobile-First Scrollable) */}
				<nav aria-label='Project views navigation'>
					<div
						class='flex overflow-x-auto no-scrollbar py-1 gap-1 -mb-px border-t border-base-200/60'
						role='tablist'
					>
						{tabs.map((tab) => {
							const isSelected = currentTab === tab.id
							return (
								<button
									key={tab.id}
									type='button'
									role='tab'
									aria-selected={isSelected}
									aria-label={`Switch to ${tab.label} tab`}
									onClick={() => {
										activeTab.value = tab.id
									}}
									data-testid={tab.testId}
									class={`px-3.5 py-2 text-xs md:text-sm font-medium rounded-md whitespace-nowrap transition-all ${
										isSelected
											? 'bg-base-200 text-primary font-semibold shadow-xs'
											: 'text-base-content/70 hover:text-base-content hover:bg-base-200/50'
									}`}
								>
									{tab.label}
								</button>
							)
						})}
					</div>
				</nav>
			</div>
		</header>
	)
}
