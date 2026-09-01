import { useState } from 'preact/hooks'
import type { Tier } from '../../schemas/models'
import { strategies } from '../../state/store'
import { TagBadge } from '../common/Badge'

export function StrategiesView() {
	const allStrategies = strategies.value
	const [selectedTier, setSelectedTier] = useState<Tier | 0>(0)

	const visibleStrategies = allStrategies.filter((s) =>
		selectedTier === 0 ? true : s.tier === selectedTier,
	)

	const tiers: Array<{ id: Tier | 0; label: string; desc: string }> = [
		{ id: 0, label: 'All Tiers', desc: 'Complete architecture rules' },
		{ id: 1, label: 'Tier 1', desc: 'Core Invariants & Zero-CGO' },
		{ id: 2, label: 'Tier 2', desc: 'Design Patterns & Touch UI' },
		{ id: 3, label: 'Tier 3', desc: 'Implementation Specs' },
	]

	return (
		<div class='flex-1 max-w-5xl w-full mx-auto p-4 sm:p-6 flex flex-col gap-5'>
			{/* Top Bar */}
			<div class='border-b border-base-200 pb-4'>
				<h2 class='text-xl sm:text-2xl font-bold tracking-tight text-base-content'>
					Architectural Strategies
				</h2>
				<p class='text-xs sm:text-sm text-base-content/60 mt-0.5'>
					Tiered guidelines and system design rules using progressive disclosure
				</p>
			</div>

			{/* Tier Selector Pills */}
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
					const tierColor =
						s.tier === 1
							? 'badge-primary'
							: s.tier === 2
								? 'badge-secondary'
								: 'badge-accent'

					return (
						<div
							key={s.id}
							class='card bg-base-100 border border-base-200 shadow-xs rounded-2xl p-4 sm:p-5 flex flex-col gap-3'
						>
							{/* Card Header */}
							<div class='flex items-start justify-between gap-3'>
								<div class='flex flex-col gap-1'>
									<div class='flex items-center gap-2 flex-wrap'>
										<span
											class={`badge badge-sm ${tierColor} font-bold text-xs uppercase tracking-wider`}
										>
											Tier {s.tier}
										</span>
										<span class='font-mono text-xs text-base-content/40'>
											#{s.id}
										</span>
									</div>
									<h3 class='text-base font-bold text-base-content mt-1 leading-snug'>
										{s.title}
									</h3>
								</div>
							</div>

							{/* Summary */}
							{s.summary && (
								<p class='text-xs sm:text-sm text-base-content/80 leading-relaxed'>
									{s.summary}
								</p>
							)}

							{/* Expandable Markdown Spec */}
							{s.body && s.body.trim() !== '' && (
								<details class='collapse collapse-arrow bg-base-200/40 border border-base-200 rounded-xl'>
									<summary class='collapse-title text-xs font-semibold py-2 px-3 min-h-0 text-primary'>
										View Full Specification & Guidelines
									</summary>
									<div class='collapse-content text-xs text-base-content/85 whitespace-pre-wrap font-sans leading-relaxed pt-2 border-t border-base-200/50'>
										{s.body}
									</div>
								</details>
							)}

							{/* Tags */}
							{s.tags && s.tags.length > 0 && (
								<div class='flex flex-wrap items-center gap-1 pt-1'>
									{s.tags.map((tag) => (
										<TagBadge key={tag} tag={tag} />
									))}
								</div>
							)}
						</div>
					)
				})}

				{visibleStrategies.length === 0 && (
					<div class='flex flex-col items-center justify-center py-16 text-center border-2 border-dashed border-base-300 rounded-2xl text-base-content/40'>
						<span class='text-sm font-medium'>No strategies found</span>
					</div>
				)}
			</div>
		</div>
	)
}
