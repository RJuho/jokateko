import { useMemo, useState } from 'preact/hooks'
import { glossary } from '../../state/store'
import { TagBadge } from '../common/Badge'

export function GlossaryView() {
	const allTerms = glossary.value
	const [searchQuery, setSearchQuery] = useState('')
	const [selectedLetter, setSelectedLetter] = useState<string>('All')

	const filteredTerms = useMemo(() => {
		const q = searchQuery.trim().toLowerCase()
		return allTerms.filter((term) => {
			if (q !== '') {
				const inTitle = term.title.toLowerCase().includes(q)
				const inSummary = term.summary.toLowerCase().includes(q)
				const inBody = (term.body || '').toLowerCase().includes(q)
				if (!inTitle && !inSummary && !inBody) {
					return false
				}
			}

			if (selectedLetter !== 'All') {
				if (!term.title.toUpperCase().startsWith(selectedLetter)) {
					return false
				}
			}

			return true
		})
	}, [allTerms, searchQuery, selectedLetter])

	// Available starting letters
	const availableLetters = useMemo(() => {
		const letters = new Set<string>()
		for (const t of allTerms) {
			const char = t.title.charAt(0).toUpperCase()
			if (/[A-Z]/.test(char)) {
				letters.add(char)
			}
		}
		return ['All', ...Array.from(letters).sort()]
	}, [allTerms])

	return (
		<div class='flex-1 max-w-5xl w-full mx-auto p-4 sm:p-6 flex flex-col gap-5'>
			{/* Top Bar */}
			<div class='flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-base-200 pb-4'>
				<div>
					<h2 class='text-xl sm:text-2xl font-bold tracking-tight text-base-content'>
						Project Glossary
					</h2>
					<p class='text-xs sm:text-sm text-base-content/60 mt-0.5'>
						Standardized domain definitions and terminology dictionary
					</p>
				</div>

				{/* Search Input */}
				<div class='w-full sm:w-64'>
					<input
						type='search'
						placeholder='Filter terms...'
						value={searchQuery}
						onInput={(e) =>
							setSearchQuery((e.target as HTMLInputElement).value)
						}
						class='input input-sm input-bordered w-full rounded-lg text-xs'
						aria-label='Filter glossary terms'
					/>
				</div>
			</div>

			{/* Alphabetical Jump Pills */}
			{availableLetters.length > 2 && (
				<div class='flex items-center gap-1 overflow-x-auto no-scrollbar pb-1'>
					{availableLetters.map((letter) => {
						const isSelected = selectedLetter === letter
						return (
							<button
								key={letter}
								type='button'
								onClick={() => setSelectedLetter(letter)}
								class={`px-2.5 py-1 text-xs font-semibold rounded-lg transition-all ${
									isSelected
										? 'bg-primary text-primary-content shadow-xs'
										: 'bg-base-200 text-base-content/70 hover:text-base-content'
								}`}
								aria-pressed={isSelected}
								aria-label={`Filter terms starting with ${letter}`}
							>
								{letter}
							</button>
						)
					})}
				</div>
			)}

			{/* Terms List */}
			<div class='grid grid-cols-1 md:grid-cols-2 gap-4'>
				{filteredTerms.map((term) => (
					<div
						key={term.id}
						class='card bg-base-100 border border-base-200 shadow-xs rounded-2xl p-5 flex flex-col gap-2.5'
					>
						{/* Term Header */}
						<div class='flex items-start justify-between gap-2'>
							<h3 class='text-base font-bold text-base-content leading-snug'>
								{term.title}
							</h3>
							<span class='font-mono text-[10px] text-base-content/40 shrink-0'>
								#{term.id}
							</span>
						</div>

						{/* Term Summary */}
						{term.summary && (
							<p class='text-xs sm:text-sm text-base-content/80 leading-relaxed'>
								{term.summary}
							</p>
						)}

						{/* Definition Body */}
						{term.body && term.body !== term.summary && (
							<div class='text-xs text-base-content/70 bg-base-200/40 p-3 rounded-xl border border-base-200 leading-relaxed'>
								{term.body}
							</div>
						)}

						{/* Tags */}
						{term.tags && term.tags.length > 0 && (
							<div class='flex flex-wrap items-center gap-1 pt-1 mt-auto'>
								{term.tags.map((tag) => (
									<TagBadge key={tag} tag={tag} />
								))}
							</div>
						)}
					</div>
				))}

				{filteredTerms.length === 0 && (
					<div class='col-span-full flex flex-col items-center justify-center py-16 text-center border-2 border-dashed border-base-300 rounded-2xl text-base-content/40'>
						<span class='text-sm font-medium'>No matching terms found</span>
					</div>
				)}
			</div>
		</div>
	)
}
