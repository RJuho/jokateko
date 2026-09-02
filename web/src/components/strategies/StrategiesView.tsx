import { useEffect, useState } from 'preact/hooks'
import { navigateTo } from '../../router'
import type { Tier } from '../../schemas/models'
import { activeStrategyId, filters, strategies } from '../../state/store'
import { t } from '../../utils/i18n'
import { TagBadge } from '../common/Badge'

export function StrategiesView() {
	const allStrategies = strategies.value
	const [selectedTier, setSelectedTier] = useState<Tier | 0>(0)
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
		if (activeId) {
			const target = allStrategies.find((s) => s.id === activeId)
			if (target && selectedTier !== 0 && target.tier !== selectedTier) {
				setSelectedTier(0)
			}
			setOpenedStrategyIds((prev) => new Set([...prev, activeId]))

			if (typeof document !== 'undefined') {
				requestAnimationFrame(() => {
					setTimeout(() => {
						const el = document.getElementById(`strategy-card-${activeId}`)
						el?.scrollIntoView({ behavior: 'smooth', block: 'center' })
					}, 60)
				})
			}
		}
	}, [activeStrategyId.value, allStrategies, selectedTier])

	// Strategy stays open until page is reloaded, search is used, or tags/tiers are changed
	useEffect(() => {
		setOpenedStrategyIds(new Set())
	}, [searchQuery, selectedTier, selectedTag])

	// Filter strategies by tier, tag, and search query
	const visibleStrategies = allStrategies.filter((s) => {
		// Tier filter
		if (selectedTier !== 0 && s.tier !== selectedTier) {
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

	const tiers: Array<{ id: Tier | 0; label: string; desc: string }> = [
		{ id: 0, label: t('all_tiers'), desc: 'Complete architecture rules' },
		{ id: 1, label: 'Tier 1', desc: 'Core Invariants & Zero-CGO' },
		{ id: 2, label: 'Tier 2', desc: 'Design Patterns & Touch UI' },
		{ id: 3, label: 'Tier 3', desc: 'Implementation Specs' },
	]

	function handleCardClick(id: string) {
		setOpenedStrategyIds((prev) => new Set([...prev, id]))
		navigateTo(`strategy/${id}`)
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
		<div class='flex-1 max-w-5xl w-full mx-auto p-4 sm:p-6 flex flex-col gap-5'>
			{/* Top Bar */}
			<div class='border-b border-base-200 pb-4'>
				<h2 class='text-xl sm:text-2xl font-bold tracking-tight text-base-content'>
					{t('architectural_strategies')}
				</h2>
				<p class='text-xs sm:text-sm text-base-content/60 mt-0.5'>
					{t('strategies_subtitle')}
				</p>
			</div>

			{/* Tier Selector Pills: Kept as is */}
			<div class='flex items-center gap-1.5 overflow-x-auto no-scrollbar pb-1'>
				{tiers.map((t) => {
					const isSelected = selectedTier === t.id
					return (
						<button
							key={t.id}
							type='button'
							onClick={() => setSelectedTier(t.id)}
							class={`px-3 py-1.5 rounded-xl text-xs font-semibold whitespace-nowrap transition-all flex items-center gap-1.5 ${
								isSelected
									? 'bg-primary text-primary-content shadow-xs'
									: 'bg-base-200 text-base-content/70 hover:text-base-content'
							}`}
							aria-pressed={isSelected}
							aria-label={`Filter strategies by ${t.label}`}
						>
							<span>{t.label}</span>
							<span class='text-[10px] opacity-70 hidden sm:inline'>
								({t.desc})
							</span>
						</button>
					)
				})}
			</div>

			{/* Strategies List */}
			<div class='flex flex-col gap-3.5'>
				{visibleStrategies.map((s) => {
					const isOpened = openedStrategyIds.has(s.id)
					const tierColor =
						s.tier === 1
							? 'badge-primary'
							: s.tier === 2
								? 'badge-secondary'
								: 'badge-accent'

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
										class={`badge badge-xs ${tierColor} font-bold text-[10px] uppercase tracking-wider px-1.5 py-0.5`}
									>
										Tier {s.tier}
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
