import { config, connectionStatus, mode } from '../../state/store'
import { formatBrowserDateTime } from '../../utils/date'

interface ModeIndicatorProps {
	/** 'header' aligns right in the desktop navbar, 'menu' aligns left in the mobile menu. */
	variant: 'header' | 'menu'
}

const styles = {
	header: {
		wrapper: 'flex flex-col items-end text-right select-none',
		liveRow:
			'flex items-center gap-1.5 text-xs font-medium text-base-content/80',
		liveLabel: 'text-[11px] font-mono leading-none',
		top: 'text-xs font-medium text-base-content/80',
		bottom: 'text-[10px] font-mono text-base-content/40 leading-tight mt-0.5',
	},
	menu: {
		wrapper: 'flex flex-col items-start gap-0.5 text-base-content/70',
		liveRow: 'flex items-center gap-1.5',
		liveLabel: 'text-xs font-mono font-medium',
		top: 'text-xs font-medium text-base-content/80',
		bottom: 'text-[10px] font-mono text-base-content/40',
	},
}

/** Shows live daemon connection state, or the client/static snapshot origin. */
export function ModeIndicator({ variant }: ModeIndicatorProps) {
	const s = styles[variant]
	const build = config.value.build
	const currentMode = mode.value
	const isConnected = connectionStatus.value === 'connected'

	const branchInfo = build?.branch ? (
		<span>
			{build.branch}
			{build.commit && ` (${build.commit.slice(0, 7)})`}
		</span>
	) : null

	if (currentMode === 'live') {
		return (
			<div
				class={s.wrapper}
				title={
					isConnected ? 'Live daemon connected' : 'Reconnecting to daemon...'
				}
				data-testid='mode-indicator-live'
				data-connection={connectionStatus.value}
			>
				<div class={s.liveRow}>
					<span class='relative flex size-2'>
						{isConnected && (
							<span class='absolute inline-flex size-full animate-ping rounded-full bg-success opacity-75' />
						)}
						<span
							class={`relative inline-flex size-2 rounded-full ${
								isConnected ? 'bg-success' : 'bg-warning'
							}`}
						/>
					</span>
					<span class={s.liveLabel}>
						{isConnected ? 'live' : 'reconnecting'}
					</span>
				</div>
				<div class={s.bottom}>{branchInfo ?? <span>daemon</span>}</div>
			</div>
		)
	}

	const isClient = currentMode === 'client'
	const formattedDate = formatBrowserDateTime(build?.time)
	return (
		<div
			class={s.wrapper}
			data-testid={isClient ? 'mode-indicator-client' : 'mode-indicator-static'}
		>
			<div class={s.top}>
				{isClient ? 'Client' : formattedDate || 'Offline Snapshot'}
			</div>
			<div class={s.bottom}>
				{branchInfo ?? <span>{isClient ? 'local storage' : 'snapshot'}</span>}
			</div>
		</div>
	)
}
