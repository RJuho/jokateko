import { Check, Link } from 'lucide-preact'
import { useEffect, useRef, useState } from 'preact/hooks'
import { useMermaidDiagrams } from '../../hooks/useMermaidDiagrams'
import { navigateTo } from '../../router'
import { activeGlossaryId, filters, glossary } from '../../state/store'
import { copyToClipboard } from '../../utils/clipboard'
import { t } from '../../utils/i18n'
import { matchesQuery, normalizeQuery } from '../../utils/search'
import { TagBadge } from '../common/Badge'

export function GlossaryView() {
	const allTerms = glossary.value
	const [openedTermIds, setOpenedTermIds] = useState<Set<string>>(new Set())
	const [copiedId, setCopiedId] = useState<string | null>(null)
	const [copiedLink, setCopiedLink] = useState<string | null>(null)

	const gridRef = useRef<HTMLDivElement>(null)

	const searchQuery = normalizeQuery(filters.value.searchQuery)

	// Render Mermaid diagrams in open glossary bodies and update on theme change
	useMermaidDiagrams(gridRef, [openedTermIds, activeGlossaryId.value, allTerms])

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

	// Term stays open until page is reloaded or search is used; the URL term is kept
	// open so this mount-time run does not undo a deep link on initial page load
	useEffect(() => {
		setOpenedTermIds(
			new Set(activeGlossaryId.value ? [activeGlossaryId.value] : []),
		)
	}, [searchQuery])

	const visibleTerms = allTerms.filter((term) =>
		matchesQuery(term, searchQuery),
	)

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

	async function handleCopyId(e: MouseEvent, id: string) {
		e.stopPropagation()
		if (await copyToClipboard(id)) {
			setCopiedId(id)
			setTimeout(() => setCopiedId(null), 1500)
		}
	}

	async function handleCopyLink(e: MouseEvent, id: string) {
		e.stopPropagation()
		if (typeof window !== 'undefined') {
			const url = `${window.location.origin}${window.location.pathname}#glossary/${id}`
			if (await copyToClipboard(url)) {
				setCopiedLink(id)
				setTimeout(() => setCopiedLink(null), 1500)
			}
		}
	}

	return (
		<main class='flex-1 max-w-5xl w-full mx-auto p-4 sm:p-6 flex flex-col gap-5'>
			{/* Top Bar */}
			<div class='border-b border-base-200 pb-4'>
				<h2 class='text-xl sm:text-2xl font-bold tracking-tight text-base-content'>
					{t('project_glossary')}
				</h2>
				<p class='text-xs sm:text-sm text-base-content/60 mt-0.5'>
					{t('glossary_subtitle')}
				</p>
			</div>

			{/* Terms Grid */}
			<div ref={gridRef} class='grid grid-cols-1 md:grid-cols-2 gap-4'>
				{visibleTerms.map((term) => {
					const isOpened = openedTermIds.has(term.id)
					const hasBody =
						(Boolean(term.body_html) && term.body_html?.trim() !== '')
						|| (Boolean(term.body)
							&& term.body.trim() !== ''
							&& term.body !== term.summary)

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
									<button
										type='button'
										onClick={() => handleCardClick(term.id)}
										class='text-left cursor-pointer group/title focus:outline-hidden'
									>
										<h3 class='text-base/snug font-bold text-base-content group-hover/title:text-primary transition-colors'>
											{term.title}
										</h3>
									</button>

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
											<Check class='size-3.5 text-success' />
										) : (
											<Link class='size-3.5' />
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
									<p class='text-xs/relaxed text-base-content/80 sm:text-sm'>
										{term.summary}
									</p>
								)}
							</button>

							{/* Expandable Body: Opens below summary with line divider */}
							{isOpened && hasBody && (
								<div class='pt-0.5'>
									<hr class='border-base-200 my-1.5' />
									{term.body_html ? (
										<div
											class='prose prose-sm max-w-none text-base-content/90 prose-headings:text-base-content prose-headings:font-bold prose-p:text-base-content/85 prose-strong:text-base-content prose-code:text-primary prose-code:bg-base-200/60 prose-code:px-1 prose-code:py-0.5 prose-code:rounded prose-code:before:content-none prose-code:after:content-none prose-pre:bg-base-200 prose-pre:text-base-content'
											dangerouslySetInnerHTML={{ __html: term.body_html }}
										/>
									) : (
										<div class='prose prose-sm max-w-none text-base-content/90 prose-p:text-base-content/85'>
											<p class='whitespace-pre-wrap'>{term.body}</p>
										</div>
									)}
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
						<span class='text-sm font-medium'>
							{t('no_terms_found', 'No glossary terms found')}
						</span>
					</div>
				)}
			</div>
		</main>
	)
}
