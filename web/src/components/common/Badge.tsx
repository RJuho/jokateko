import type { ComponentChildren } from 'preact'
import type { Priority } from '../../schemas/models'

interface PriorityBadgeProps {
	priority: Priority
	className?: string
}

export function PriorityBadge({
	priority,
	className = '',
}: PriorityBadgeProps) {
	const colorMap: Record<Priority, string> = {
		critical: 'badge-error text-error-content',
		high: 'badge-warning text-warning-content',
		medium: 'badge-info text-info-content',
		low: 'badge-neutral text-neutral-content',
	}

	return (
		<span
			class={`badge badge-sm font-semibold uppercase tracking-wider text-[10px] ${colorMap[priority]} ${className}`}
			role='status'
			aria-label={`Priority: ${priority}`}
			data-testid='task-priority-badge'
		>
			{priority}
		</span>
	)
}

interface TagBadgeProps {
	tag: string
	selected?: boolean
	onClick?: () => void
	className?: string
}

export function TagBadge({
	tag,
	selected = false,
	onClick,
	className = '',
}: TagBadgeProps) {
	const activeClass = selected
		? 'badge-primary font-medium shadow-xs'
		: 'badge-outline hover:badge-primary transition-colors cursor-pointer'

	if (onClick) {
		return (
			<button
				type='button'
				onClick={onClick}
				class={`badge badge-sm text-xs transition-all ${activeClass} ${className}`}
				aria-pressed={selected}
				aria-label={`Filter by tag ${tag}`}
			>
				#{tag}
			</button>
		)
	}

	return (
		<span class={`badge badge-sm badge-outline text-xs ${className}`}>
			#{tag}
		</span>
	)
}

interface ModeBadgeProps {
	mode: 'live' | 'static'
	connectionStatus?: 'connected' | 'connecting' | 'disconnected'
	className?: string
}

export function ModeBadge({
	mode,
	connectionStatus,
	className = '',
}: ModeBadgeProps) {
	if (mode === 'live') {
		const isConnected = connectionStatus === 'connected'
		return (
			<span
				class={`badge badge-sm badge-success gap-1.5 font-medium ${className}`}
				role='status'
				aria-label='Live daemon connected'
				data-testid='mode-indicator-live'
			>
				<span
					class={`inline-block w-1.5 h-1.5 rounded-full ${
						isConnected ? 'bg-success-content animate-pulse' : 'bg-warning'
					}`}
					aria-hidden='true'
				/>
				Live
			</span>
		)
	}

	return (
		<span
			class={`badge badge-sm badge-neutral gap-1.5 font-medium ${className}`}
			role='status'
			aria-label='Static snapshot read-only'
			data-testid='mode-indicator-static'
		>
			<span
				class='inline-block w-1.5 h-1.5 rounded-full bg-base-300'
				aria-hidden='true'
			/>
			Static
		</span>
	)
}

interface GenericBadgeProps {
	children: ComponentChildren
	variant?: 'primary' | 'secondary' | 'accent' | 'neutral' | 'outline'
	size?: 'xs' | 'sm' | 'md'
	className?: string
}

export function GenericBadge({
	children,
	variant = 'neutral',
	size = 'sm',
	className = '',
}: GenericBadgeProps) {
	const variantClass =
		variant === 'outline' ? 'badge-outline' : `badge-${variant}`
	const sizeClass = size === 'xs' ? 'text-[10px] h-4' : `badge-${size}`

	return (
		<span class={`badge ${variantClass} ${sizeClass} ${className}`}>
			{children}
		</span>
	)
}
