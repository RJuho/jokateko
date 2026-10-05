import { TriangleAlert, X } from 'lucide-preact'
import { useState } from 'preact/hooks'
import { validationWarnings } from '../../state/store'
import { t } from '../../utils/i18n'

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
				<TriangleAlert class='mt-0.5 size-5 shrink-0' />
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
					aria-label={
						isExpanded
							? t('arial_hide_issue_details')
							: t('arial_show_issue_details')
					}
				>
					{isExpanded ? 'Hide Details' : 'View Details'}
				</button>
				<button
					type='button'
					onClick={() => {
						validationWarnings.value = []
					}}
					class='btn btn-xs btn-ghost btn-circle'
					aria-label={t('arial_dismiss_schema_warning')}
				>
					<X class='size-4' />
				</button>
			</div>
		</div>
	)
}
