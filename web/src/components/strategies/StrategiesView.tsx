import { useEffect, useState } from 'preact/hooks'
import { navigateTo } from '../../router'
import {
	activeStrategyId,
	configuredTiers,
	filters,
	strategies,
} from '../../state/store'
import { getContrastTextColor } from '../../utils/colors'
import { t } from '../../utils/i18n'
import { TagBadge } from '../common/Badge'

export function StrategiesView() {
	const allStrategies = strategies.value
	const tierList = configuredTiers.value
	const [selectedTier, setSelectedTier] = useState<string | number>(0)
	const [selectedTag, setSelectedTag] = useState<string | null>(null)
	const [openedStrategyIds, setOpenedStrategyIds] = useState<Set<string>>(
		new Set(),
	)
	const [copiedId, setCopiedId] = useState<string | null>(null)
	const [copiedLink, setCopiedLink] = useState<string | null>(null)

	const searchQuery = filters.value.searchQuery.trim().toLowerCase()

	// If initial or current URL anchor has a strategy, ensure it is open and scroll into it
	useEffect(() => {
		const activeId = activeStrategyId.value
		if (!activeId || allStrategies.length === 0) {
			return
		}

		const target = allStrategies.find((s) => s.id === activeId)
		if (!target) {
			return
		}

		// Ensure targeted strategy is visible by resetting tier filter if currently filtering another tier
		if (selectedTier !== 0 && String(target.tier) !== String(selectedTier)) {
			setSelectedTier(0)
		}

		// Ensure targeted card is recorded in opened set
		setOpenedStrategyIds((prev) => new Set([...prev, activeId]))

		// Scroll smoothly into view with retries to account for DOM rendering
		if (typeof document !== 'undefined') {
			let attempts = 0
			const maxAttempts = 15
			const tryScroll = () => {
				const el = document.getElementById(`strategy-card-${activeId}`)
				if (el) {
					el.scrollIntoView({ behavior: 'smooth', block: 'center' })
				} else if (attempts < maxAttempts) {
					attempts++
					setTimeout(tryScroll, 60)
				}
			}
			requestAnimationFrame(() => {
				setTimeout(tryScroll, 60)
			})
		}
	}, [activeStrategyId.value, allStrategies.length])

	// Close non-URL opened strategies when search or tag filter changes
	useEffect(() => {
		if (searchQuery !== '' || selectedTag !== null) {
			setOpenedStrategyIds(
				new Set(activeStrategyId.value ? [activeStrategyId.value] : []),
			)
		}
	}, [searchQuery, selectedTag])

	// Filter strategies by tier, tag, and search query
	const visibleStrategies = allStrategies.filter((s) => {
		// Tier filter
		if (selectedTier !== 0 && String(s.tier) !== String(selectedTier)) {
			return false
		}
		// Tag filter
		if (selectedTag !== null && !s.tags?.includes(selectedTag)) {
			return false
		}
		// Search query
		if (searchQuery !== '') {
			const matchesTitle = s.title.toLowerCase().includes(searchQuery)
			const matchesSummary = s.summary?.toLowerCase().includes(searchQuery)
			const matchesId = s.id.toLowerCase().includes(searchQuery)
			const matchesTags = s.tags?.some((t) =>
				t.toLowerCase().includes(searchQuery),
			)
			const matchesBody = s.body?.toLowerCase().includes(searchQuery)
			if (
				!matchesTitle
				&& !matchesSummary
				&& !matchesId
				&& !matchesTags
				&& !matchesBody
			) {
				return false
			}
		}
		return true
	})

	const tiers: Array<{
		id: string | number
		label: string
		title: string
		summary: string
		color?: string
	}> = [
		{
			id: 0,
			label: t('all_tiers'),
			title: 'Complete architecture rules',
			summary:
				'View all strategy guidelines across every progressive disclosure tier',
		},
		...tierList.map((tr) => ({
			id: tr.id,
			label: tr.name,
			title: tr.title,
			summary: tr.summary,
			color: tr.color,
		})),
	]

	function handleSelectTier(tierId: string | number) {
		setSelectedTier(tierId)
		// If an active strategy from URL is not in this tier, navigate back to #strategies
		const activeId = activeStrategyId.value
		if (activeId && tierId !== 0) {
			const target = allStrategies.find((s) => s.id === activeId)
			if (target && String(target.tier) !== String(tierId)) {
				activeStrategyId.value = null
				navigateTo('strategies', true)
			}
		}
	}

	function handleCardClick(id: string) {
		const isCurrentlyActive = activeStrategyId.value === id
		const isCurrentlyOpened = openedStrategyIds.has(id) || isCurrentlyActive

		if (isCurrentlyOpened) {
			setOpenedStrategyIds((prev) => {
				const next = new Set(prev)
				next.delete(id)
				return next
			})
			if (isCurrentlyActive) {
				activeStrategyId.value = null
				navigateTo('strategies', true)
			}
		} else {
			setOpenedStrategyIds((prev) => new Set([...prev, id]))
			navigateTo(`strategy/${id}`)
		}
	}

	function handleCopyId(e: MouseEvent, id: string) {
		e.stopPropagation()
		navigator.clipboard.writeText(id)
		setCopiedId(id)
		setTimeout(() => setCopiedId(null), 1500)
	}

	function handleCopyLink(e: MouseEvent, id: string) {
		e.stopPropagation()
		if (typeof window !== 'undefined') {
			const url = `${window.location.origin}${window.location.pathname}#strategy/${id}`
			navigator.clipboard.writeText(url)
			setCopiedLink(id)
			setTimeout(() => setCopiedLink(null), 1500)
		}
	}

	function toggleTagFilter(tag: string) {
		setSelectedTag((prev) => (prev === tag ? null : tag))
	}

	return (
		<div class='flex flex-col gap-5 max-w-5xl mx-auto py-2 px-1 sm:px-2'>
			{/* Page Header */}
			<div class='flex flex-col gap-1 border-b border-base-200/80 pb-4'>
				<h1 class='text-2xl font-black tracking-tight text-base-content'>
					{t('architectural_strategies')}
				</h1>
				<p class='text-sm text-base-content/60 leading-relaxed'>
					{t('strategies_subtitle')}
				</p>
			</div>

			{/* Tier Selector Pills: Kept as is */}
			<div class='flex items-center gap-1.5 overflow-x-auto no-scrollbar pb-1'>
				{tiers.map((tr) => {
					const isSelected = String(selectedTier) === String(tr.id)
					const color = tr.color
					const activeStyle =
						isSelected && color
							? {
									backgroundColor: color,
									color: getContrastTextColor(color),
								}
							: undefined

					return (
						<button
							key={tr.id}
							type='button'
							onClick={() => handleSelectTier(tr.id)}
							class={`px-3 py-1.5 rounded-xl text-xs font-semibold whitespace-nowrap transition-all flex items-center gap-1.5 ${
								isSelected
									? color
										? 'shadow-xs'
										: 'bg-primary text-primary-content shadow-xs'
									: 'bg-base-200 text-base-content/70 hover:text-base-content'
							}`}
							style={activeStyle}
							title={tr.summary || tr.title}
							aria-pressed={isSelected}
							aria-label={`Filter strategies by ${tr.label}`}
						>
							<span>{tr.label}</span>
							<span class='text-[10px] opacity-70 hidden sm:inline'>
								({tr.title})
							</span>
						</button>
					)
				})}
			</div>

			{/* Strategies List */}
			<div class='flex flex-col gap-3.5'>
				{visibleStrategies.map((s) => {
					const isOpened =
						openedStrategyIds.has(s.id) || activeStrategyId.value === s.id
					const tierCfg = tierList.find(
						(tr) => String(tr.id) === String(s.tier),
					)
					const tierLabel = tierCfg?.name || `Tier ${s.tier}`
					const tierColor = tierCfg?.color
					const badgeStyle = tierColor
						? {
								backgroundColor: tierColor,
								color: getContrastTextColor(tierColor),
							}
						: undefined

					return (
						<div
							key={s.id}
							id={`strategy-card-${s.id}`}
							class={`card bg-base-100 border border-base-200 shadow-xs hover:border-primary/40 transition-all rounded-2xl p-4 sm:p-5 flex flex-col gap-2.5 ${
								isOpened ? 'ring-1 ring-primary/20 bg-base-100/90' : ''
							}`}
							data-testid={`strategy-card-${s.id}`}
						>
							{/* Card Top Row: Tier badge (xs), Strategy ID (no #, click to copy), Link icon button */}
							<div class='flex items-center justify-between gap-2'>
								<div class='flex items-center gap-2 flex-wrap'>
									{/* Tier Badge: xs */}
									<span
										class={`badge badge-xs font-bold text-[10px] uppercase tracking-wider px-1.5 py-0.5 shadow-2xs ${
											!tierColor ? 'badge-primary text-primary-content' : ''
										}`}
										style={badgeStyle}
										title={
											tierCfg ? `${tierCfg.name}: ${tierCfg.title}` : undefined
										}
										data-testid='strategy-tier-badge'
									>
										{tierLabel}
									</span>

									{/* Strategy ID: Click to copy without "#" */}
									<button
										type='button'
										onClick={(e) => handleCopyId(e, s.id)}
										class='font-mono text-xs text-base-content/50 hover:text-base-content hover:bg-base-200/60 px-1.5 py-0.5 rounded transition-colors'
										title={
											copiedId === s.id
												? 'Copied to clipboard!'
												: 'Click to copy ID'
										}
										aria-label={`Copy strategy ID ${s.id}`}
									>
										{copiedId === s.id ? 'copied!' : s.id}
									</button>

									{/* Link Icon Button: Copies anchor link */}
									<button
										type='button'
										onClick={(e) => handleCopyLink(e, s.id)}
										class='p-1 rounded text-base-content/40 hover:text-base-content hover:bg-base-200/60 transition-colors'
										title={
											copiedLink === s.id
												? 'Link copied to clipboard!'
												: 'Copy link to strategy'
										}
										aria-label={`Copy link to strategy ${s.id}`}
									>
										{copiedLink === s.id ? (
											<svg
												class='w-3.5 h-3.5 text-success'
												fill='none'
												viewBox='0 0 24 24'
												stroke='currentColor'
												stroke-width='2'
												aria-hidden='true'
											>
												<title>Copied</title>
												<path
													stroke-linecap='round'
													stroke-linejoin='round'
													d='M5 13l4 4L19 7'
												/>
											</svg>
										) : (
											<svg
												class='w-3.5 h-3.5'
												fill='none'
												viewBox='0 0 24 24'
												stroke='currentColor'
												stroke-width='2'
												aria-hidden='true'
											>
												<title>Copy link</title>
												<path
													stroke-linecap='round'
													stroke-linejoin='round'
													d='M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1'
												/>
											</svg>
										)}
									</button>
								</div>
							</div>

							{/* Clickable Area: Title, Tags, Summary */}
							<button
								type='button'
								onClick={() => handleCardClick(s.id)}
								class='w-full text-left flex flex-col gap-2 cursor-pointer focus:outline-hidden group'
								aria-expanded={isOpened}
								aria-label={`Strategy: ${s.title}`}
							>
								{/* Title */}
								<h3 class='text-base font-bold text-base-content leading-snug group-hover:text-primary transition-colors'>
									{s.title}
								</h3>

								{/* Tags under Tier badge and title, like on board page, no "Tags:" label */}
								{s.tags && s.tags.length > 0 && (
									<div class='flex flex-wrap items-center gap-1'>
										{s.tags.map((tag) => (
											<TagBadge
												key={tag}
												tag={tag}
												selected={selectedTag === tag}
												onClick={() => toggleTagFilter(tag)}
											/>
										))}
									</div>
								)}

								{/* Summary: Kept as is */}
								{s.summary && (
									<p class='text-xs sm:text-sm text-base-content/80 leading-relaxed pt-0.5'>
										{s.summary}
									</p>
								)}
							</button>

							{/* Expandable Body: Opens below summary with line divider, formatted like in Task modal */}
							{isOpened && s.body && s.body.trim() !== '' && (
								<div class='pt-1'>
									<hr class='border-base-200 my-1.5' />
									<div class='text-xs sm:text-sm text-base-content/85 space-y-2 font-sans'>
										{s.body.split('\n').map((line, idx) => {
											const trimmed = line.trim()
											if (!trimmed) {
												return <div key={`empty-${idx}`} class='h-1' />
											}
											if (trimmed.startsWith('#')) {
												const headingText = trimmed.replace(/^#+\s*/, '')
												return (
													<h4
														key={`h-${idx}`}
														class='font-bold text-sm sm:text-base text-base-content pt-2'
													>
														{headingText}
													</h4>
												)
											}
											if (
												trimmed.startsWith('- ')
												|| trimmed.startsWith('* ')
											) {
												return (
													<div
														key={`li-${idx}`}
														class='flex items-start gap-2 pl-2'
													>
														<span class='text-primary font-bold'>•</span>
														<span>{trimmed.slice(2)}</span>
													</div>
												)
											}
											return (
												<p
													key={`p-${idx}`}
													class='leading-relaxed whitespace-pre-wrap'
												>
													{line}
												</p>
											)
										})}
									</div>
								</div>
							)}
						</div>
					)
				})}

				{visibleStrategies.length === 0 && (
					<div class='flex flex-col items-center justify-center py-16 text-center border-2 border-dashed border-base-300 rounded-2xl text-base-content/40'>
						<span class='text-sm font-medium'>
							{t('no_strategies_found', 'No strategies found')}
						</span>
					</div>
				)}
			</div>
		</div>
	)
}
