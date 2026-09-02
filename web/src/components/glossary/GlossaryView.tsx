import { useEffect, useState } from 'preact/hooks'
import { navigateTo } from '../../router'
import { activeGlossaryId, filters, glossary } from '../../state/store'
import { TagBadge } from '../common/Badge'

export function GlossaryView() {
	const allTerms = glossary.value
	const [openedTermIds, setOpenedTermIds] = useState<Set<string>>(new Set())
	const [copiedId, setCopiedId] = useState<string | null>(null)
	const [copiedLink, setCopiedLink] = useState<string | null>(null)

	const searchQuery = filters.value.searchQuery.trim().toLowerCase()

	// If initial or current URL anchor has a term, open its body and scroll into view
	useEffect(() => {
		const activeId = activeGlossaryId.value
		if (activeId) {
			setOpenedTermIds((prev) => new Set([...prev, activeId]))

			if (typeof document !== 'undefined') {
				requestAnimationFrame(() => {
					setTimeout(() => {
						const el = document.getElementById(`glossary-card-${activeId}`)
						el?.scrollIntoView({ behavior: 'smooth', block: 'center' })
					}, 60)
				})
			}
		}
	}, [activeGlossaryId.value])

	// Term stays open until page is reloaded or search is used
	useEffect(() => {
		setOpenedTermIds(new Set())
	}, [searchQuery])

	const visibleTerms = allTerms.filter((term) => {
		if (searchQuery === '') {
			return true
		}
		const inTitle = term.title.toLowerCase().includes(searchQuery)
		const inId = term.id.toLowerCase().includes(searchQuery)
		const inSummary = term.summary.toLowerCase().includes(searchQuery)
		const inBody = (term.body || '').toLowerCase().includes(searchQuery)
		const inTags =
			term.tags?.some((t) => t.toLowerCase().includes(searchQuery)) ?? false
		return inTitle || inId || inSummary || inBody || inTags
	})

	function handleCardClick(id: string) {
		setOpenedTermIds((prev) => {
			const next = new Set(prev)
			if (next.has(id)) {
				next.delete(id)
			} else {
				next.add(id)
			}
			return next
		})
		navigateTo(`glossary/${id}`)
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
			const url = `${window.location.origin}${window.location.pathname}#glossary/${id}`
			navigator.clipboard.writeText(url)
			setCopiedLink(id)
			setTimeout(() => setCopiedLink(null), 1500)
		}
	}

	return (
		<div class='flex-1 max-w-5xl w-full mx-auto p-4 sm:p-6 flex flex-col gap-5'>
			{/* Top Bar */}
			<div class='border-b border-base-200 pb-4'>
				<h2 class='text-xl sm:text-2xl font-bold tracking-tight text-base-content'>
					Project Glossary
				</h2>
				<p class='text-xs sm:text-sm text-base-content/60 mt-0.5'>
					Standardized domain definitions and terminology dictionary
				</p>
			</div>

			{/* Terms Grid */}
			<div class='grid grid-cols-1 md:grid-cols-2 gap-4'>
				{visibleTerms.map((term) => {
					const isOpened = openedTermIds.has(term.id)
					const hasBody =
						Boolean(term.body)
						&& term.body.trim() !== ''
						&& term.body !== term.summary

					return (
						<div
							key={term.id}
							id={`glossary-card-${term.id}`}
							class={`card bg-base-100 border border-base-200 shadow-xs hover:border-primary/40 transition-all rounded-2xl p-5 flex flex-col gap-2.5 ${
								isOpened ? 'ring-1 ring-primary/20 bg-base-100/90' : ''
							}`}
							data-testid={`glossary-card-${term.id}`}
						>
							{/* Term Header: Title + Link button on left, Term ID (no '#', click to copy) on right */}
							<div class='flex items-start justify-between gap-2'>
								<div class='flex items-center gap-1.5 min-w-0'>
									<h3 class='text-base font-bold text-base-content leading-snug'>
										{term.title}
									</h3>

									{/* Link Button next to title */}
									<button
										type='button'
										onClick={(e) => handleCopyLink(e, term.id)}
										class='p-1 rounded text-base-content/40 hover:text-base-content hover:bg-base-200/60 transition-colors shrink-0'
										title={
											copiedLink === term.id
												? 'Link copied to clipboard!'
												: 'Copy link to term'
										}
										aria-label={`Copy link to glossary term ${term.title}`}
									>
										{copiedLink === term.id ? (
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

								{/* Term ID: Click to copy without '#' */}
								<button
									type='button'
									onClick={(e) => handleCopyId(e, term.id)}
									class='font-mono text-[10px] text-base-content/40 hover:text-base-content hover:bg-base-200/60 px-1.5 py-0.5 rounded transition-colors shrink-0'
									title={
										copiedId === term.id
											? 'Copied to clipboard!'
											: 'Click to copy ID'
									}
									aria-label={`Copy glossary ID ${term.id}`}
								>
									{copiedId === term.id ? 'copied!' : term.id}
								</button>
							</div>

							{/* Clickable Card Area: Summary */}
							<button
								type='button'
								onClick={() => handleCardClick(term.id)}
								class='w-full text-left flex flex-col gap-2 cursor-pointer focus:outline-hidden group'
								aria-expanded={isOpened}
								aria-label={`Glossary term: ${term.title}`}
							>
								{/* Term Summary */}
								{term.summary && (
									<p class='text-xs sm:text-sm text-base-content/80 leading-relaxed'>
										{term.summary}
									</p>
								)}
							</button>

							{/* Expandable Body: Opens below summary with line divider */}
							{isOpened && hasBody && (
								<div class='pt-0.5'>
									<hr class='border-base-200 my-1.5' />
									<div class='text-xs text-base-content/85 space-y-2 font-sans'>
										{term.body.split('\n').map((line, idx) => {
											const trimmed = line.trim()
											if (!trimmed) {
												return <div key={`empty-${idx}`} class='h-1' />
											}
											if (trimmed.startsWith('#')) {
												const headingText = trimmed.replace(/^#+\s*/, '')
												return (
													<h4
														key={`h-${idx}`}
														class='font-bold text-xs sm:text-sm text-base-content pt-1'
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

							{/* Tags: Kept where they are (pt-1 mt-auto) */}
							{term.tags && term.tags.length > 0 && (
								<div class='flex flex-wrap items-center gap-1 pt-1 mt-auto'>
									{term.tags.map((tag) => (
										<TagBadge key={tag} tag={tag} />
									))}
								</div>
							)}
						</div>
					)
				})}

				{visibleTerms.length === 0 && (
					<div class='col-span-full flex flex-col items-center justify-center py-16 text-center border-2 border-dashed border-base-300 rounded-2xl text-base-content/40'>
						<span class='text-sm font-medium'>No glossary terms found</span>
					</div>
				)}
			</div>
		</div>
	)
}
