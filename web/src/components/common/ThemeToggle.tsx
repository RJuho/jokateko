import { Moon, Sun } from 'lucide-preact'
import {
	DARK_THEME,
	isDarkTheme,
	LIGHT_THEME,
	setTheme,
} from '../../state/theme'
import { t } from '../../utils/i18n'

interface ThemeToggleProps {
	testId?: string
}

/** Sun/moon swap toggling between the light and dark daisyUI themes. */
export function ThemeToggle({ testId }: ThemeToggleProps) {
	return (
		<label
			class='swap swap-rotate btn btn-ghost btn-sm btn-square text-base-content/70 hover:text-base-content'
			aria-label={t('arial_theme_toggle')}
		>
			<input
				type='checkbox'
				name='theme'
				class='theme-controller'
				value={DARK_THEME}
				checked={isDarkTheme.value}
				onChange={(e) =>
					setTheme(
						(e.target as HTMLInputElement).checked ? DARK_THEME : LIGHT_THEME,
					)
				}
				aria-label={t('arial_theme_dark')}
				data-testid={testId}
			/>

			{/* sun icon (swap-off: shown when light theme is active) */}
			<Sun class='size-4 swap-off'>
				<title>{t('arial_theme_light_label')}</title>
			</Sun>

			{/* moon icon (swap-on: shown when dark theme is active) */}
			<Moon class='size-4 swap-on'>
				<title>{t('arial_theme_dark_label')}</title>
			</Moon>
		</label>
	)
}
