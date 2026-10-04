import { computed, effect, signal } from '@preact/signals'

export const DARK_THEME = 'sunset'
export const LIGHT_THEME = 'winter'
const THEME_STORAGE_KEY = 'theme'

const darkThemes = new Set([
	'sunset',
	'dark',
	'synthwave',
	'halloween',
	'forest',
	'aqua',
	'black',
	'luxury',
	'dracula',
	'business',
	'night',
	'coffee',
	'dim',
	'abyss',
])

export const theme = signal<string>(DARK_THEME)
export const isDarkTheme = computed(() => darkThemes.has(theme.value))

function readSavedTheme(): string | null {
	try {
		return localStorage.getItem(THEME_STORAGE_KEY)
	} catch {
		return null
	}
}

function systemPrefersDark(): boolean {
	if (typeof window === 'undefined' || !window.matchMedia) {
		return true
	}
	return window.matchMedia('(prefers-color-scheme: dark)').matches
}

/** Applies a theme chosen by the user and remembers it across reloads. */
export function setTheme(next: string): void {
	theme.value = next
	try {
		localStorage.setItem(THEME_STORAGE_KEY, next)
	} catch {
		// storage unavailable: theme still applies for this session
	}
}

/**
 * Resolves the initial theme (saved choice, else OS preference), keeps the
 * `data-theme` attribute in sync with the `theme` signal, and follows OS
 * preference changes until the user picks a theme explicitly.
 */
export function initTheme(): () => void {
	if (typeof document === 'undefined') {
		return () => {}
	}
	const root = document.documentElement
	theme.value =
		readSavedTheme() ?? (systemPrefersDark() ? DARK_THEME : LIGHT_THEME)

	const stopEffect = effect(() => {
		if (root.getAttribute('data-theme') !== theme.value) {
			root.setAttribute('data-theme', theme.value)
		}
	})

	// Respect external attribute changes (devtools, tests, other scripts)
	const observer = new MutationObserver(() => {
		const attr = root.getAttribute('data-theme')
		if (attr && attr !== theme.value) {
			theme.value = attr
		}
	})
	observer.observe(root, { attributes: true, attributeFilter: ['data-theme'] })

	const mediaQuery = window.matchMedia?.('(prefers-color-scheme: dark)')
	const handleSystemChange = (e: MediaQueryListEvent) => {
		if (readSavedTheme()) {
			return
		}
		theme.value = e.matches ? DARK_THEME : LIGHT_THEME
	}
	mediaQuery?.addEventListener?.('change', handleSystemChange)

	return () => {
		stopEffect()
		observer.disconnect()
		mediaQuery?.removeEventListener?.('change', handleSystemChange)
	}
}
