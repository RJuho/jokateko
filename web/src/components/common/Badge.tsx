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
			class={`badge badge-xs font-semibold uppercase tracking-wider text-[9px] px-1.5 py-0.5 ${colorMap[priority]} ${className}`}
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
				{tag}
			</button>
		)
	}

	return (
		<span
			class={`badge badge-sm badge-ghost text-xs ${className}`}
			data-testid={`tag-${tag}`}
		>
			{tag}
		</span>
	)
}

interface MilestoneBadgeProps {
	milestone: string
	className?: string
}

export function MilestoneBadge({
	milestone,
	className = '',
}: MilestoneBadgeProps) {
	return (
		<span
			class={`badge badge-xs badge-ghost text-[10px] font-medium text-base-content/70 ${className}`}
			data-testid={`milestone-${milestone}`}
		>
			{milestone}
		</span>
	)
}

interface StatusPillProps {
	status: string
	children?: ComponentChildren
	className?: string
}

export function StatusPill({
	status,
	children,
	className = '',
}: StatusPillProps) {
	return (
		<span
			class={`badge badge-sm font-semibold uppercase tracking-wider ${className}`}
		>
			{children || status}
		</span>
	)
}
