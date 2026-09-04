import mermaid from 'mermaid'

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

export function isDarkTheme(): boolean {
	if (typeof document === 'undefined') return false
	const currentTheme = document.documentElement.getAttribute('data-theme') || ''
	if (currentTheme) {
		return darkThemes.has(currentTheme)
	}
	return (
		typeof window !== 'undefined'
		&& (window.matchMedia?.('(prefers-color-scheme: dark)').matches ?? false)
	)
}

let mermaidInitialized = false
let currentTheme: 'dark' | 'default' | null = null

export function configureMermaid(forceTheme?: 'dark' | 'default') {
	const theme = forceTheme || (isDarkTheme() ? 'dark' : 'default')
	if (mermaidInitialized && currentTheme === theme) {
		return
	}
	mermaid.initialize({
		startOnLoad: false,
		securityLevel: 'strict',
		theme,
		fontFamily: 'inherit',
		suppressErrorRendering: true,
	})
	mermaidInitialized = true
	currentTheme = theme
}

let diagramSeq = 0

/**
 * Renders all code.language-mermaid blocks within the given container element into SVG diagrams.
 * If a diagram fails to parse/render, the original <pre><code> block is preserved untouched.
 */
export async function renderMermaidDiagrams(
	container: HTMLElement,
): Promise<void> {
	if (!container) return

	const mermaidBlocks = container.querySelectorAll<HTMLElement>(
		'pre > code.language-mermaid',
	)
	if (mermaidBlocks.length === 0) return

	configureMermaid()

	for (const codeEl of Array.from(mermaidBlocks)) {
		const preEl = codeEl.parentElement
		if (!preEl) continue

		// Store raw source code on the pre element so theme switches can re-render it
		const rawCode =
			preEl.getAttribute('data-mermaid-source') || codeEl.textContent || ''
		if (!rawCode.trim()) continue

		preEl.setAttribute('data-mermaid-source', rawCode)

		const diagramId = `mermaid-svg-${Date.now()}-${++diagramSeq}`
		try {
			// Validate syntax with parse first
			await mermaid.parse(rawCode)
			const { svg } = await mermaid.render(diagramId, rawCode)

			const wrapper = document.createElement('div')
			wrapper.className =
				'mermaid-diagram my-3 overflow-x-auto flex justify-center p-2 rounded-xl bg-base-200/30'
			wrapper.setAttribute('data-testid', 'mermaid-diagram')
			wrapper.setAttribute('data-mermaid-source', rawCode)
			wrapper.innerHTML = svg

			preEl.replaceWith(wrapper)
		} catch (err) {
			console.warn(
				'[Mermaid] Failed to parse diagram, retaining raw markdown block:',
				err,
			)
			// Retain preEl untouched as graceful fallback
		}
	}
}

/**
 * Re-renders existing Mermaid diagrams inside container (e.g. when theme changes).
 */
export async function reRenderMermaidDiagrams(
	container: HTMLElement,
): Promise<void> {
	if (!container) return

	// Reset any rendered diagrams back to pre elements
	const rendered = container.querySelectorAll<HTMLElement>(
		'.mermaid-diagram[data-mermaid-source]',
	)
	for (const el of Array.from(rendered)) {
		const raw = el.getAttribute('data-mermaid-source') || ''
		const pre = document.createElement('pre')
		pre.setAttribute('data-mermaid-source', raw)
		const code = document.createElement('code')
		code.className = 'language-mermaid'
		code.textContent = raw
		pre.appendChild(code)
		el.replaceWith(pre)
	}

	// Force re-config with new theme
	currentTheme = null
	configureMermaid()
	await renderMermaidDiagrams(container)
}
