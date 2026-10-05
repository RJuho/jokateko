import { Calendar } from 'lucide-preact'
import type { ComponentChildren } from 'preact'
import type { Priority } from '../../schemas/models'
import { configuredPriorities } from '../../state/store'
import { getContrastTextColor } from '../../utils/colors'
import { t, tf, uiLocale } from '../../utils/i18n'
import { isDoneStatus } from '../../utils/status'

interface PriorityBadgeProps {
	priority: Priority
	className?: string
}

export function PriorityBadge({
	priority,
	className = '',
}: PriorityBadgeProps) {
	const priorityCfg = configuredPriorities.value.find((p) => p.id === priority)
	const label = priorityCfg?.name || priority
	const color = priorityCfg?.color

	const style = color
		? {
				backgroundColor: color,
				color: getContrastTextColor(color),
			}
		: undefined

	return (
		<span
			class={`badge badge-xs font-semibold uppercase tracking-wider text-[9px] px-1.5 py-0.5 shadow-2xs ${
				!color ? 'badge-neutral text-neutral-content' : ''
			} ${className}`}
			style={style}
			role='status'
			aria-label={tf('arial_priority', { priority: label })}
			data-testid='task-priority-badge'
		>
			{label}
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
		? 'badge-soft badge-primary font-medium shadow-xs'
		: 'badge-ghost hover:bg-base-300 transition-colors cursor-pointer'

	if (onClick) {
		return (
			<button
				type='button'
				onClick={onClick}
				class={`badge badge-sm text-xs transition-all ${activeClass} ${className}`}
				aria-pressed={selected}
				aria-label={tf('arial_filter_by_tag', { tag })}
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

interface TargetDateBadgeProps {
	targetAt?: string | null
	status?: string
	className?: string
}

export function TargetDateBadge({
	targetAt,
	status,
	className = '',
}: TargetDateBadgeProps) {
	if (!targetAt) return null

	const date = new Date(targetAt)
	if (Number.isNaN(date.getTime())) return null

	const formatted = date.toLocaleDateString(uiLocale(), {
		month: 'short',
		day: 'numeric',
		year:
			date.getFullYear() !== new Date().getFullYear() ? 'numeric' : undefined,
	})

	const isDone = isDoneStatus(status)
	let badgeClass = 'badge-ghost text-base-content/70'
	let titleKey:
		| 'arial_target_title'
		| 'arial_target_overdue_title'
		| 'arial_target_approaching_title' = 'arial_target_title'

	if (!isDone) {
		const today = new Date()
		today.setHours(0, 0, 0, 0)
		const targetDay = new Date(date)
		targetDay.setHours(0, 0, 0, 0)
		const diffDays = Math.ceil(
			(targetDay.getTime() - today.getTime()) / (1000 * 60 * 60 * 24),
		)

		if (diffDays < 0) {
			badgeClass = 'badge-error text-error-content'
			titleKey = 'arial_target_overdue_title'
		} else if (diffDays <= 3) {
			badgeClass = 'badge-warning text-warning-content'
			titleKey = 'arial_target_approaching_title'
		}
	} else {
		badgeClass = 'badge-ghost text-base-content/40'
	}

	return (
		<span
			class={`badge badge-xs gap-1 font-medium text-[10px] px-1.5 py-0.5 ${badgeClass} ${className}`}
			title={tf(titleKey, { date: targetAt })}
			data-testid='target-date-badge'
		>
			<Calendar class='size-3'>
				<title>{t('arial_target_date')}</title>
			</Calendar>
			<span>{formatted}</span>
		</span>
	)
}
