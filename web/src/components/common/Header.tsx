import { Info, Menu, Search, X } from 'lucide-preact'
import { useEffect, useMemo, useRef, useState } from 'preact/hooks'
import { navigateTo } from '../../router'
import type {
	GlossaryTerm,
	Milestone,
	Strategy,
	Task,
} from '../../schemas/models'
import {
	activeTab,
	config,
	displayVersion,
	filters,
	glossary,
	milestones,
	openAboutModal,
	setSearchQuery,
	strategies,
	type Tab,
	tasks,
} from '../../state/store'
import { t } from '../../utils/i18n'
import { matchesQuery, normalizeQuery } from '../../utils/search'
import { ModeIndicator } from './ModeIndicator'
import { ThemeToggle } from './ThemeToggle'

interface SearchResultItem {
	id: string
	type: 'task' | 'strategy' | 'glossary' | 'milestone'
	tab: Tab
	title: string
	snippet: string
	badge: string
}

function computeSearchResults(
	query: string,
	currentTab: Tab,
	allTasks: Task[],
	allStrategies: Strategy[],
	allGlossary: GlossaryTerm[],
	allMilestones: Milestone[],
): {
	currentPageResults: SearchResultItem[]
	otherPagesResults: SearchResultItem[]
} {
	const q = normalizeQuery(query)
	if (!q) {
		return { currentPageResults: [], otherPagesResults: [] }
	}

	const taskResults: SearchResultItem[] = allTasks
		.filter((t) => matchesQuery(t, q))
		.map((t) => ({
			id: t.id,
			type: 'task',
			tab: 'board',
			title: t.title,
			snippet: t.summary || t.body || t.id,
			badge: 'Task',
		}))

	const strategyResults: SearchResultItem[] = allStrategies
		.filter((s) => matchesQuery(s, q))
		.map((s) => ({
			id: s.id,
			type: 'strategy',
			tab: 'strategies',
			title: s.title,
			snippet: s.summary || s.body || s.id,
			badge: `Tier ${s.tier}`,
		}))

	const glossaryResults: SearchResultItem[] = allGlossary
		.filter((g) => matchesQuery(g, q))
		.map((g) => ({
			id: g.id,
			type: 'glossary',
			tab: 'glossary',
			title: g.title,
			snippet: g.summary || g.body || g.id,
			badge: 'Term',
		}))

	// Milestones open on the board, filtered by the selected milestone
	const milestoneResults: SearchResultItem[] = allMilestones
		.filter((m) => matchesQuery(m, q))
		.map((m) => ({
			id: m.id,
			type: 'milestone',
			tab: 'board',
			title: m.title,
			snippet: m.summary || m.body || m.id,
			badge: m.status === 'closed' ? 'Closed' : 'Milestone',
		}))

	let currentPageAll: SearchResultItem[] = []
	let otherPagesAll: SearchResultItem[] = []

	if (currentTab === 'board') {
		currentPageAll = taskResults
		otherPagesAll = [
			...milestoneResults,
			...strategyResults,
			...glossaryResults,
		]
	} else if (currentTab === 'calendar') {
		currentPageAll = [...taskResults, ...milestoneResults]
		otherPagesAll = [...strategyResults, ...glossaryResults]
	} else if (currentTab === 'strategies') {
		currentPageAll = strategyResults
		otherPagesAll = [...taskResults, ...milestoneResults, ...glossaryResults]
	} else {
		currentPageAll = glossaryResults
		otherPagesAll = [...taskResults, ...milestoneResults, ...strategyResults]
	}

	return {
		currentPageResults: currentPageAll.slice(0, 2),
		otherPagesResults: otherPagesAll.slice(0, 2),
	}
}

export function Header() {
	const currentTab = activeTab.value
	const projectName = config.value.project?.name || 'Jokateko'
	const version = displayVersion(config.value.build)
	const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false)
	const [isSearchDropdownOpen, setIsSearchDropdownOpen] = useState(false)

	const searchInputRef = useRef<HTMLInputElement>(null)
	const searchContainerRef = useRef<HTMLDivElement>(null)

	const isMac =
		typeof navigator !== 'undefined'
		&& /Mac|iPhone|iPod|iPad/i.test(navigator.userAgent)

	const navTabs: { id: Tab; label: string; testId: string }[] = [
		{ id: 'board', label: t('board'), testId: 'nav-tab-board' },
		{ id: 'calendar', label: t('calendar'), testId: 'nav-tab-calendar' },
		{ id: 'strategies', label: t('strategies'), testId: 'nav-tab-strategies' },
		{ id: 'glossary', label: t('glossary'), testId: 'nav-tab-glossary' },
	]

	// Search results
	const { currentPageResults, otherPagesResults } = useMemo(() => {
		return computeSearchResults(
			filters.value.searchQuery,
			currentTab,
			tasks.value,
			strategies.value,
			glossary.value,
			milestones.value,
		)
	}, [
		filters.value.searchQuery,
		currentTab,
		tasks.value,
		strategies.value,
		glossary.value,
		milestones.value,
	])

	const hasAnyResults =
		currentPageResults.length > 0 || otherPagesResults.length > 0

	// Global shortcut for CMD/CTRL + K and click outside listener
	useEffect(() => {
		function handleGlobalKeyDown(e: KeyboardEvent) {
			if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
				e.preventDefault()
				searchInputRef.current?.focus()
				searchInputRef.current?.select()
				if (filters.value.searchQuery.trim().length > 0) {
					setIsSearchDropdownOpen(true)
				}
			}
		}

		function handleClickOutside(e: MouseEvent) {
			if (
				searchContainerRef.current
				&& !searchContainerRef.current.contains(e.target as Node)
			) {
				setIsSearchDropdownOpen(false)
			}
		}

		window.addEventListener('keydown', handleGlobalKeyDown)
		document.addEventListener('mousedown', handleClickOutside)
		return () => {
			window.removeEventListener('keydown', handleGlobalKeyDown)
			document.removeEventListener('mousedown', handleClickOutside)
		}
	}, [])

	function handleTabClick(tabId: Tab) {
		navigateTo(tabId)
		setIsMobileMenuOpen(false)
		setIsSearchDropdownOpen(false)
	}

	function handleSearchInput(e: Event) {
		const val = (e.target as HTMLInputElement).value
		setSearchQuery(val)
		setIsSearchDropdownOpen(val.trim().length > 0)
	}

	function handleSearchKeyDown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			e.stopPropagation()
			setIsSearchDropdownOpen(false)
		}
	}

	function handleSelectResult(item: SearchResultItem) {
		if (item.type === 'task') {
			navigateTo(`task/${item.id}`)
		} else if (item.type === 'strategy') {
			navigateTo(`strategy/${item.id}`)
		} else if (item.type === 'glossary') {
			navigateTo(`glossary/${item.id}`)
		} else if (item.type === 'milestone') {
			setSearchQuery('')
			if (searchInputRef.current) {
				searchInputRef.current.value = ''
				searchInputRef.current.blur()
			}
			navigateTo(`milestone/${item.id}`)
		}
		setIsSearchDropdownOpen(false)
	}

	return (
		<header class='sticky top-0 z-30 bg-base-100/90 backdrop-blur-md border-b border-base-200/80 shadow-xs'>
			{/* No container: wide as possible across the entire screen */}
			<div class='w-full px-3 sm:px-4 md:px-6'>
				<div class='flex items-center h-12 md:h-13 gap-1.5'>
					{/* 1. Desktop Brand Name: config.project.name (just text, not button nor link) */}
					<span
						class='hidden md:inline-flex font-bold text-sm sm:text-base tracking-tight shrink-0 select-none'
						data-testid='navbar-brand-name'
					>
						{projectName}
					</span>

					{/* 2. Desktop Navigation Buttons: Board, Strategies, Glossary (hidden on mobile) */}
					<nav
						class='hidden md:flex items-center gap-1.5 shrink-0'
						aria-label={t('arial_main_nav')}
					>
						{navTabs.map((tab) => {
							const isSelected = currentTab === tab.id
							return (
								<button
									key={tab.id}
									type='button'
									onClick={() => handleTabClick(tab.id)}
									data-testid={tab.testId}
									class={`btn btn-sm border-none ${
										isSelected ? 'btn-soft btn-primary' : 'btn-ghost'
									}`}
									aria-current={isSelected ? 'page' : undefined}
								>
									{tab.label}
								</button>
							)
						})}
					</nav>

					{/* 3. Search Textfield using daisyUI label.input (higher input height, uniform gap with no margin) */}
					<search
						ref={searchContainerRef}
						class='relative flex-1 min-w-0'
						aria-label={t('arial_search')}
					>
						<label class='input input-sm w-full h-9 sm:h-9 md:h-10 text-xs sm:text-sm flex items-center'>
							<Search
								class='h-[1em] w-auto opacity-50 shrink-0'
								strokeWidth={2.5}
							/>
							<input
								ref={searchInputRef}
								id='global-search'
								name='q'
								type='search'
								class='grow text-xs focus:outline-hidden bg-transparent'
								placeholder={t('search')}
								value={filters.value.searchQuery}
								onInput={handleSearchInput}
								onKeyDown={handleSearchKeyDown}
								onFocus={() => {
									if (filters.value.searchQuery.trim().length > 0) {
										setIsSearchDropdownOpen(true)
									}
								}}
								aria-label={t('arial_search_input')}
								data-testid='search-input'
							/>
							{isMac ? (
								<>
									<kbd class='hidden sm:inline-flex kbd kbd-xs select-none'>
										⌘
									</kbd>
									<kbd class='hidden sm:inline-flex kbd kbd-xs select-none'>
										K
									</kbd>
								</>
							) : (
								<>
									<kbd class='hidden sm:inline-flex kbd kbd-xs select-none'>
										Ctrl
									</kbd>
									<kbd class='hidden sm:inline-flex kbd kbd-xs select-none'>
										K
									</kbd>
								</>
							)}
						</label>

						{/* Quick Search Results Popover */}
						{isSearchDropdownOpen
							&& filters.value.searchQuery.trim().length > 0 && (
								<section
									class='animate-fadeIn absolute inset-x-0 top-full z-50 mt-1.5 flex flex-col gap-1.5 rounded-2xl border border-base-200 bg-base-100/98 p-2 shadow-2xl backdrop-blur-md'
									data-testid='search-results-popover'
									aria-label={t('arial_search_results')}
									onKeyDown={(e) => {
										if (e.key === 'Escape') {
											e.stopPropagation()
											setIsSearchDropdownOpen(false)
											searchInputRef.current?.focus()
										}
									}}
								>
									{/* 1. Max two results from current page */}
									<div class='flex flex-col gap-0.5'>
										{currentPageResults.length > 0 ? (
											currentPageResults.map((item) => (
												<button
													key={`${item.type}-${item.id}`}
													type='button'
													onClick={() => handleSelectResult(item)}
													class='w-full text-left p-2 rounded-xl hover:bg-base-200/70 focus:bg-base-200/90 focus:outline-hidden transition-all flex flex-col gap-0.5 group cursor-pointer'
													tabIndex={0}
												>
													<div class='flex items-center justify-between gap-2'>
														<span class='font-bold text-xs text-base-content group-hover:text-primary transition-colors truncate'>
															{item.title}
														</span>
														<span class='badge badge-xs badge-ghost text-[9px] shrink-0'>
															{item.badge}
														</span>
													</div>
													{item.snippet && (
														<p class='text-[11px] text-base-content/65 line-clamp-1 leading-snug'>
															{item.snippet}
														</p>
													)}
												</button>
											))
										) : (
											<div class='px-2 py-1 text-[11px] text-base-content/40 italic'>
												{t('no_matches_current_page')}
											</div>
										)}
									</div>

									{/* 2. Under two current page results, max two mini results from other pages */}
									<div class='border-t border-base-200/70 pt-1 flex flex-col gap-0.5'>
										{otherPagesResults.length > 0 ? (
											otherPagesResults.map((item) => (
												<button
													key={`${item.type}-${item.id}`}
													type='button'
													onClick={() => handleSelectResult(item)}
													class='w-full text-left px-2 py-1.5 rounded-xl hover:bg-base-200/70 focus:bg-base-200/90 focus:outline-hidden transition-all flex items-center justify-between gap-2 group cursor-pointer'
													tabIndex={0}
												>
													<div class='flex flex-col gap-0.5 min-w-0'>
														<span class='font-medium text-xs text-base-content group-hover:text-primary transition-colors truncate'>
															{item.title}
														</span>
														{item.snippet && (
															<p class='text-[10px] text-base-content/50 line-clamp-1'>
																{item.snippet}
															</p>
														)}
													</div>
													<span class='badge badge-xs badge-outline text-[9px] shrink-0 uppercase tracking-wider'>
														{item.tab}
													</span>
												</button>
											))
										) : (
											<div class='px-2 py-1 text-[11px] text-base-content/40 italic'>
												{t('no_matches_other_pages')}
											</div>
										)}
									</div>

									{!hasAnyResults && (
										<div class='p-2 text-center text-xs text-base-content/50'>
											{t('no_matching_results')}
										</div>
									)}
								</section>
							)}
					</search>

					{/* Theme Controller using a swap (sun/moon toggle) for desktop */}
					<div class='hidden md:flex items-center shrink-0'>
						<ThemeToggle testId='theme-controller-toggle' />
					</div>

					{/* About & Licenses Trigger */}
					<div class='hidden md:flex items-center shrink-0'>
						<button
							type='button'
							onClick={() => openAboutModal('about')}
							class='btn btn-ghost btn-sm btn-square text-base-content/70 hover:text-base-content'
							title='About & Licenses'
							aria-label='About Jokateko and Open Source Licenses'
							data-testid='about-modal-trigger'
						>
							<Info class='size-4' />
						</button>
					</div>

					{/* 4. Information Box: Datetime / Live status stacked ON TOP of branch & commit */}
					<div class='hidden md:flex items-center shrink-0'>
						<ModeIndicator variant='header' />
					</div>

					{/* 5. Mobile Hamburger Menu Toggle (< md: on the right side of navbar) */}
					<button
						type='button'
						onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
						class='md:hidden btn btn-ghost btn-sm btn-square shrink-0'
						aria-label={
							isMobileMenuOpen ? t('arial_close_menu') : t('arial_open_menu')
						}
						aria-expanded={isMobileMenuOpen}
						data-testid='mobile-menu-toggle'
					>
						{isMobileMenuOpen ? <X class='size-5' /> : <Menu class='size-5' />}
					</button>
				</div>

				{/* Mobile Dropdown Panel (< md) */}
				{isMobileMenuOpen && (
					<section
						class='md:hidden py-3 px-2 border-t border-base-200/80 flex flex-col gap-2.5 bg-base-100/98 shadow-lg animate-fadeIn'
						aria-label={t('arial_mobile_menu')}
					>
						{/* Mobile Brand Name Header & Theme Toggle */}
						<div class='flex items-center justify-between pb-2 border-b border-base-200/60 px-1'>
							<span
								class='font-bold text-base tracking-tight px-1 select-none'
								data-testid='mobile-menu-brand-name'
							>
								{projectName}
							</span>

							<ThemeToggle />
						</div>

						{/* Navigation Tabs */}
						<nav class='flex flex-col gap-1' aria-label={t('arial_mobile_nav')}>
							{navTabs.map((tab) => {
								const isSelected = currentTab === tab.id
								return (
									<button
										key={tab.id}
										type='button'
										onClick={() => {
											handleTabClick(tab.id)
											setIsMobileMenuOpen(false)
										}}
										data-testid={tab.testId}
										class={`btn btn-sm border-none justify-start ${
											isSelected ? 'btn-soft btn-primary' : 'btn-ghost'
										}`}
										aria-current={isSelected ? 'page' : undefined}
									>
										{tab.label}
									</button>
								)
							})}
							<button
								type='button'
								onClick={() => {
									setIsMobileMenuOpen(false)
									openAboutModal('about')
								}}
								data-testid='mobile-about-modal-trigger'
								class='btn btn-sm btn-ghost justify-start gap-2 text-base-content/80'
							>
								<Info class='size-4' />
								<span>About & Licenses</span>
							</button>
						</nav>

						{/* Mobile Information Box: Datetime / Live on top of branch */}
						<div class='pt-2 border-t border-base-200/60 px-1'>
							<ModeIndicator variant='menu' />
						</div>

						{/* Mobile Footer Attribution & Version: inserted under server status / build info with small font */}
						<div class='pt-2 border-t border-base-200/60 px-1 flex items-center justify-between text-[11px] text-base-content/55 select-none'>
							<span>{t('footer_text')}</span>
							<div class='flex items-center gap-1.5'>
								<a
									href='https://github.com/RJuho/jokateko'
									target='_blank'
									rel='noopener noreferrer'
									class='link link-hover font-medium text-base-content/75 hover:text-primary'
									aria-label={t('arial_github_repo')}
								>
									Jokateko
								</a>
								<span class='badge badge-xs badge-ghost font-mono opacity-80'>
									{version}
								</span>
							</div>
						</div>
					</section>
				)}
			</div>
		</header>
	)
}
