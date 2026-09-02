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
	connectionStatus,
	filters,
	glossary,
	milestones,
	mode,
	setSearchQuery,
	strategies,
	type Tab,
	tasks,
} from '../../state/store'
import { formatBrowserDateTime } from '../../utils/date'
import { t } from '../../utils/i18n'

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
	const q = query.trim().toLowerCase()
	if (!q) {
		return { currentPageResults: [], otherPagesResults: [] }
	}

	const taskResults: SearchResultItem[] = allTasks
		.filter((t) => {
			const inTitle = t.title.toLowerCase().includes(q)
			const inId = t.id.toLowerCase().includes(q)
			const inSummary = (t.summary || '').toLowerCase().includes(q)
			const inBody = (t.body || '').toLowerCase().includes(q)
			const inTags =
				t.tags?.some((tag) => tag.toLowerCase().includes(q)) ?? false
			return inTitle || inId || inSummary || inBody || inTags
		})
		.map((t) => ({
			id: t.id,
			type: 'task',
			tab: 'board',
			title: t.title,
			snippet: t.summary || t.body || t.id,
			badge: 'Task',
		}))

	const strategyResults: SearchResultItem[] = allStrategies
		.filter((s) => {
			const inTitle = s.title.toLowerCase().includes(q)
			const inId = s.id.toLowerCase().includes(q)
			const inSummary = (s.summary || '').toLowerCase().includes(q)
			const inBody = (s.body || '').toLowerCase().includes(q)
			const inTags =
				s.tags?.some((tag) => tag.toLowerCase().includes(q)) ?? false
			return inTitle || inId || inSummary || inBody || inTags
		})
		.map((s) => ({
			id: s.id,
			type: 'strategy',
			tab: 'strategies',
			title: s.title,
			snippet: s.summary || s.body || s.id,
			badge: `Tier ${s.tier}`,
		}))

	const glossaryResults: SearchResultItem[] = allGlossary
		.filter((g) => {
			const inTitle = g.title.toLowerCase().includes(q)
			const inId = g.id.toLowerCase().includes(q)
			const inSummary = (g.summary || '').toLowerCase().includes(q)
			const inBody = (g.body || '').toLowerCase().includes(q)
			const inTags =
				g.tags?.some((tag) => tag.toLowerCase().includes(q)) ?? false
			return inTitle || inId || inSummary || inBody || inTags
		})
		.map((g) => ({
			id: g.id,
			type: 'glossary',
			tab: 'glossary',
			title: g.title,
			snippet: g.summary || g.body || g.id,
			badge: 'Term',
		}))

	const milestoneResults: SearchResultItem[] = allMilestones
		.filter((m) => {
			const inTitle = m.title.toLowerCase().includes(q)
			const inId = m.id.toLowerCase().includes(q)
			const inSummary = (m.summary || '').toLowerCase().includes(q)
			const inBody = (m.body || '').toLowerCase().includes(q)
			const inTags =
				m.tags?.some((tag) => tag.toLowerCase().includes(q)) ?? false
			return inTitle || inId || inSummary || inBody || inTags
		})
		.map((m) => ({
			id: m.id,
			type: 'milestone',
			tab: 'milestones',
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
	} else if (currentTab === 'milestones') {
		currentPageAll = milestoneResults
		otherPagesAll = [...taskResults, ...strategyResults, ...glossaryResults]
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
	const isLive = mode.value === 'live'
	const buildInfo = config.value.build
	const projectName = config.value.project?.name || 'Jokateko'
	const version =
		buildInfo?.version && buildInfo.version !== 'dev'
			? buildInfo.version
			: '0.1.0'
	const displayVersion = version.startsWith('v') ? version : `v${version}`
	const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false)
	const [isSearchDropdownOpen, setIsSearchDropdownOpen] = useState(false)
	const [isDark, setIsDark] = useState(true)

	const DARK_THEME = 'sunset'
	const LIGHT_THEME = 'winter'

	const searchInputRef = useRef<HTMLInputElement>(null)
	const searchContainerRef = useRef<HTMLDivElement>(null)

	const formattedDate = formatBrowserDateTime(buildInfo?.time)

	// Initialize theme and listen to system preference changes (e.g. evening / morning OS changes)
	useEffect(() => {
		if (typeof window === 'undefined') return

		const mediaQuery = window.matchMedia?.('(prefers-color-scheme: dark)')

		const savedTheme = localStorage.getItem('theme')
		let initialTheme: string
		if (savedTheme) {
			initialTheme = savedTheme
		} else {
			const prefersDark = mediaQuery?.matches ?? true
			initialTheme = prefersDark ? DARK_THEME : LIGHT_THEME
		}

		setIsDark(initialTheme === DARK_THEME)
		document.documentElement.setAttribute('data-theme', initialTheme)

		function handleSystemChange(e: MediaQueryListEvent) {
			const newTheme = e.matches ? DARK_THEME : LIGHT_THEME
			setIsDark(e.matches)
			document.documentElement.setAttribute('data-theme', newTheme)
			localStorage.setItem('theme', newTheme)
		}

		mediaQuery?.addEventListener?.('change', handleSystemChange)
		return () => {
			mediaQuery?.removeEventListener?.('change', handleSystemChange)
		}
	}, [])

	function handleThemeToggle(e: Event) {
		const checked = (e.target as HTMLInputElement).checked
		setIsDark(checked)
		const theme = checked ? DARK_THEME : LIGHT_THEME
		document.documentElement.setAttribute('data-theme', theme)
		localStorage.setItem('theme', theme)
	}

	const isMac =
		typeof navigator !== 'undefined'
		&& /Mac|iPhone|iPod|iPad/i.test(navigator.userAgent)

	const navTabs: { id: Tab; label: string; testId: string }[] = [
		{ id: 'board', label: t('board'), testId: 'nav-tab-board' },
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
							<svg
								class='h-[1em] opacity-50 shrink-0'
								xmlns='http://www.w3.org/2000/svg'
								viewBox='0 0 24 24'
								aria-hidden='true'
							>
								<g
									stroke-linejoin='round'
									stroke-linecap='round'
									stroke-width='2.5'
									fill='none'
									stroke='currentColor'
								>
									<circle cx='11' cy='11' r='8' />
									<path d='m21 21-4.3-4.3' />
								</g>
							</svg>
							<input
								ref={searchInputRef}
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
									<kbd class='hidden sm:inline-flex kbd kbd-xs text-[10px] select-none'>
										⌘
									</kbd>
									<kbd class='hidden sm:inline-flex kbd kbd-xs text-[10px] select-none'>
										K
									</kbd>
								</>
							) : (
								<>
									<kbd class='hidden sm:inline-flex kbd kbd-xs text-[10px] select-none'>
										Ctrl
									</kbd>
									<kbd class='hidden sm:inline-flex kbd kbd-xs text-[10px] select-none'>
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
						<label
							class='swap swap-rotate btn btn-ghost btn-sm btn-square text-base-content/70 hover:text-base-content'
							aria-label={t('arial_theme_toggle')}
						>
							<input
								type='checkbox'
								class='theme-controller'
								value={DARK_THEME}
								checked={isDark}
								onChange={handleThemeToggle}
								aria-label={t('arial_theme_dark')}
								data-testid='theme-controller-toggle'
							/>

							{/* sun icon (swap-off: shown when winter / light theme is active) */}
							<svg
								class='size-4 swap-off fill-current'
								xmlns='http://www.w3.org/2000/svg'
								viewBox='0 0 24 24'
								aria-hidden='true'
							>
								<title>{t('arial_theme_light_label')}</title>
								<path d='M5.64,17l-.71.71a1,1,0,0,0,0,1.41,1,1,0,0,0,1.41,0l.71-.71A1,1,0,0,0,5.64,17ZM5,12a1,1,0,0,0-1-1H3a1,1,0,0,0,0,2H4A1,1,0,0,0,5,12Zm7-7a1,1,0,0,0,1-1V3a1,1,0,0,0-2,0V4A1,1,0,0,0,12,5ZM5.64,7.05a1,1,0,0,0,.7.29,1,1,0,0,0,.71-.29,1,1,0,0,0,0-1.41l-.71-.71A1,1,0,0,0,4.93,6.34Zm12,.29a1,1,0,0,0,.7-.29l.71-.71a1,1,0,1,0-1.41-1.41L17,5.64a1,1,0,0,0,0,1.41A1,1,0,0,0,17.66,7.34ZM21,11H20a1,1,0,0,0,0,2h1a1,1,0,0,0,0-2Zm-9,8a1,1,0,0,0-1,1v1a1,1,0,0,0,2,0V20A1,1,0,0,0,12,19ZM18.36,17A1,1,0,0,0,17,18.36l.71.71a1,1,0,0,0,1.41,0,1,1,0,0,0,0-1.41ZM12,6.5A5.5,5.5,0,1,0,17.5,12,5.51,5.51,0,0,0,12,6.5Zm0,9A3.5,3.5,0,1,1,15.5,12,3.5,3.5,0,0,1,12,15.5Z' />
							</svg>

							{/* moon icon (swap-on: shown when sunset / dark theme is active) */}
							<svg
								class='size-4 swap-on fill-current'
								xmlns='http://www.w3.org/2000/svg'
								viewBox='0 0 24 24'
								aria-hidden='true'
							>
								<title>{t('arial_theme_dark_label')}</title>
								<path d='M21.64,13a1,1,0,0,0-1.05-.14,8.05,8.05,0,0,1-3.37.73A8.15,8.15,0,0,1,9.08,5.49a8.59,8.59,0,0,1,.25-2A1,1,0,0,0,8,2.36,10.14,10.14,0,1,0,22,14.05A1,1,0,0,0,21.64,13Zm-9.5,6.69A8.14,8.14,0,0,1,7.08,5.22v.27A10.15,10.15,0,0,0,17.22,15.63a9.79,9.79,0,0,0,2.1-.22A8.11,8.11,0,0,1,12.14,19.73Z' />
							</svg>
						</label>
					</div>

					{/* 4. Information Box: Datetime / Live status stacked ON TOP of branch & commit */}
					<div class='hidden md:flex items-center shrink-0'>
						{isLive ? (
							/* Live Mode: Top = Live indicator, Bottom = Branch / Daemon info */
							<div
								class='flex flex-col items-end text-right select-none'
								title={
									connectionStatus.value === 'connected'
										? 'Live daemon connected'
										: 'Reconnecting to daemon...'
								}
								data-testid='mode-indicator-live'
							>
								<div class='flex items-center gap-1.5 text-xs font-medium text-base-content/80'>
									<span class='relative flex size-2'>
										<span class='absolute inline-flex size-full animate-ping rounded-full bg-emerald-400 opacity-75' />
										<span class='relative inline-flex size-2 rounded-full bg-emerald-500' />
									</span>
									<span class='text-[11px] font-mono leading-none'>
										{connectionStatus.value === 'connected'
											? 'live'
											: 'reconnecting'}
									</span>
								</div>
								<div class='text-[10px] font-mono text-base-content/40 leading-tight mt-0.5'>
									{buildInfo?.branch ? (
										<span>
											{buildInfo.branch}
											{buildInfo.commit && ` (${buildInfo.commit.slice(0, 7)})`}
										</span>
									) : (
										<span>daemon</span>
									)}
								</div>
							</div>
						) : (
							/* Client / Static Mode */
							<div
								class='flex flex-col items-end text-right select-none'
								data-testid={
									mode.value === 'client'
										? 'mode-indicator-client'
										: 'mode-indicator-static'
								}
							>
								<div class='text-xs font-medium text-base-content/80'>
									{mode.value === 'client'
										? 'Client'
										: formattedDate || 'Offline Snapshot'}
								</div>
								<div class='text-[10px] font-mono text-base-content/40 leading-tight mt-0.5'>
									{buildInfo?.branch ? (
										<span>
											{buildInfo.branch}
											{buildInfo.commit && ` (${buildInfo.commit.slice(0, 7)})`}
										</span>
									) : (
										<span>
											{mode.value === 'client' ? 'local storage' : 'snapshot'}
										</span>
									)}
								</div>
							</div>
						)}
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
						<svg
							xmlns='http://www.w3.org/2000/svg'
							class='size-5'
							fill='none'
							viewBox='0 0 24 24'
							stroke='currentColor'
							aria-hidden='true'
						>
							<path
								stroke-linecap='round'
								stroke-linejoin='round'
								stroke-width='2'
								d={
									isMobileMenuOpen
										? 'M6 18L18 6M6 6l12 12'
										: 'M4 6h16M4 12h16M4 18h16'
								}
							/>
						</svg>
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

							<label
								class='swap swap-rotate btn btn-ghost btn-sm btn-square text-base-content/70 hover:text-base-content'
								aria-label={t('arial_theme_toggle')}
							>
								<input
									type='checkbox'
									class='theme-controller'
									value={DARK_THEME}
									checked={isDark}
									onChange={handleThemeToggle}
									aria-label={t('arial_theme_dark')}
								/>
								<svg
									class='size-4 swap-off fill-current'
									xmlns='http://www.w3.org/2000/svg'
									viewBox='0 0 24 24'
									aria-hidden='true'
								>
									<title>{t('arial_theme_light_label')}</title>
									<path d='M5.64,17l-.71.71a1,1,0,0,0,0,1.41,1,1,0,0,0,1.41,0l.71-.71A1,1,0,0,0,5.64,17ZM5,12a1,1,0,0,0-1-1H3a1,1,0,0,0,0,2H4A1,1,0,0,0,5,12Zm7-7a1,1,0,0,0,1-1V3a1,1,0,0,0-2,0V4A1,1,0,0,0,12,5ZM5.64,7.05a1,1,0,0,0,.7.29,1,1,0,0,0,.71-.29,1,1,0,0,0,0-1.41l-.71-.71A1,1,0,0,0,4.93,6.34Zm12,.29a1,1,0,0,0,.7-.29l.71-.71a1,1,0,1,0-1.41-1.41L17,5.64a1,1,0,0,0,0,1.41A1,1,0,0,0,17.66,7.34ZM21,11H20a1,1,0,0,0,0,2h1a1,1,0,0,0,0-2Zm-9,8a1,1,0,0,0-1,1v1a1,1,0,0,0,2,0V20A1,1,0,0,0,12,19ZM18.36,17A1,1,0,0,0,17,18.36l.71.71a1,1,0,0,0,1.41,0,1,1,0,0,0,0-1.41ZM12,6.5A5.5,5.5,0,1,0,17.5,12,5.51,5.51,0,0,0,12,6.5Zm0,9A3.5,3.5,0,1,1,15.5,12,3.5,3.5,0,0,1,12,15.5Z' />
								</svg>
								<svg
									class='size-4 swap-on fill-current'
									xmlns='http://www.w3.org/2000/svg'
									viewBox='0 0 24 24'
									aria-hidden='true'
								>
									<title>{t('arial_theme_dark_label')}</title>
									<path d='M21.64,13a1,1,0,0,0-1.05-.14,8.05,8.05,0,0,1-3.37.73A8.15,8.15,0,0,1,9.08,5.49a8.59,8.59,0,0,1,.25-2A1,1,0,0,0,8,2.36,10.14,10.14,0,1,0,22,14.05A1,1,0,0,0,21.64,13Zm-9.5,6.69A8.14,8.14,0,0,1,7.08,5.22v.27A10.15,10.15,0,0,0,17.22,15.63a9.79,9.79,0,0,0,2.1-.22A8.11,8.11,0,0,1,12.14,19.73Z' />
								</svg>
							</label>
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
						</nav>

						{/* Mobile Information Box: Datetime / Live on top of branch */}
						<div class='pt-2 border-t border-base-200/60 px-1'>
							{isLive ? (
								<div
									class='flex flex-col items-start gap-0.5 text-base-content/70'
									data-testid='mode-indicator-live'
								>
									<div class='flex items-center gap-1.5'>
										<span class='relative flex size-2'>
											<span class='absolute inline-flex size-full animate-ping rounded-full bg-emerald-400 opacity-75' />
											<span class='relative inline-flex size-2 rounded-full bg-emerald-500' />
										</span>
										<span class='text-xs font-mono font-medium'>
											{connectionStatus.value === 'connected'
												? 'live'
												: 'reconnecting'}
										</span>
									</div>
									<div class='text-[10px] font-mono text-base-content/40'>
										{buildInfo?.branch ? (
											<span>
												{buildInfo.branch}
												{buildInfo.commit
													&& ` (${buildInfo.commit.slice(0, 7)})`}
											</span>
										) : (
											<span>daemon</span>
										)}
									</div>
								</div>
							) : (
								<div
									class='flex flex-col items-start gap-0.5 font-mono text-base-content/70'
									data-testid={
										mode.value === 'client'
											? 'mode-indicator-client'
											: 'mode-indicator-static'
									}
								>
									<div class='text-xs font-medium text-base-content/80'>
										{mode.value === 'client'
											? 'Client'
											: formattedDate || 'Offline Snapshot'}
									</div>
									<div class='text-[10px] text-base-content/40'>
										{buildInfo?.branch ? (
											<span>
												{buildInfo.branch}
												{buildInfo.commit
													&& ` (${buildInfo.commit.slice(0, 7)})`}
											</span>
										) : (
											<span>
												{mode.value === 'client' ? 'local storage' : 'snapshot'}
											</span>
										)}
									</div>
								</div>
							)}
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
									{displayVersion}
								</span>
							</div>
						</div>
					</section>
				)}
			</div>
		</header>
	)
}
