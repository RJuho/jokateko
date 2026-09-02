import { useState } from 'preact/hooks'
import { validationWarnings } from '../../state/store'

export function ValidationBanner() {
	const warnings = validationWarnings.value
	const [isExpanded, setIsExpanded] = useState(false)

	if (warnings.length === 0) {
		return null
	}

	return (
		<div
			class='alert alert-warning shadow-sm mb-4 text-sm flex flex-col md:flex-row items-start md:items-center justify-between gap-2'
			role='alert'
			aria-live='polite'
			data-testid='validation-error-banner'
		>
			<div class='flex items-start gap-2'>
				<svg
					xmlns='http://www.w3.org/2000/svg'
					class='mt-0.5 size-5 shrink-0 stroke-current'
					fill='none'
					viewBox='0 0 24 24'
					aria-hidden='true'
				>
					<path
						stroke-linecap='round'
						stroke-linejoin='round'
						stroke-width='2'
						d='M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z'
					/>
				</svg>
				<div>
					<span class='font-bold'>Schema Warning: </span>
					<span>
						Found {warnings.length} issue{warnings.length > 1 ? 's' : ''} in
						snapshot data. Rendering available fields.
					</span>

					{isExpanded && (
						<ul class='mt-2 list-disc list-inside text-xs space-y-1 bg-warning-content/10 p-2 rounded-sm max-h-40 overflow-y-auto'>
							{warnings.map((w, idx) => (
								<li key={idx} class='font-mono'>
									{w}
								</li>
							))}
						</ul>
					)}
				</div>
			</div>

			<div class='flex items-center gap-2 self-end md:self-auto'>
				<button
					type='button'
					onClick={() => setIsExpanded(!isExpanded)}
					class='btn btn-xs btn-ghost'
					aria-expanded={isExpanded}
					aria-label={isExpanded ? 'Hide issue details' : 'Show issue details'}
				>
					{isExpanded ? 'Hide Details' : 'View Details'}
				</button>
				<button
					type='button'
					onClick={() => {
						validationWarnings.value = []
					}}
					class='btn btn-xs btn-ghost btn-circle'
					aria-label='Dismiss schema warning banner'
				>
					✕
				</button>
			</div>
		</div>
	)
}
