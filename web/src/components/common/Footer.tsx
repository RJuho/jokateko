import { config } from '../../state/store'
import { t } from '../../utils/i18n'

export function Footer() {
	const build = config.value.build
	const version =
		build?.version && build.version !== 'dev' ? build.version : '0.1.0'
	const displayVersion = version.startsWith('v') ? version : `v${version}`

	return (
		<footer
			class='hidden md:flex w-full h-7 shrink-0 items-center justify-between px-4 md:px-6 border-t border-base-200/80 bg-base-100 text-[11px] text-base-content/60 select-none'
			data-testid='app-footer'
			role='contentinfo'
			aria-label={t('arial_footer')}
		>
			{/* Left side: Love line */}
			<div class='flex items-center gap-1.5'>
				<span>{t('footer_text')}</span>
			</div>

			{/* Right side: Name / version */}
			<div class='flex items-center gap-2'>
				<a
					href='https://github.com/RJuho/jokateko'
					target='_blank'
					rel='noopener noreferrer'
					class='link link-hover font-medium text-base-content/80 hover:text-primary transition-colors'
					aria-label={t('arial_github_repo')}
				>
					Jokateko
				</a>
				<span class='badge badge-xs badge-ghost font-mono opacity-80'>
					{displayVersion}
				</span>
			</div>
		</footer>
	)
}
