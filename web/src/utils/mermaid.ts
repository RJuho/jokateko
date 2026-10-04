import mermaid from 'mermaid'
import { isDarkTheme } from '../state/theme'

let mermaidInitialized = false
let currentTheme: 'dark' | 'default' | null = null

export function configureMermaid(forceTheme?: 'dark' | 'default') {
	const theme = forceTheme || (isDarkTheme.peek() ? 'dark' : 'default')
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

export function openDiagramLightbox(svgHtml: string) {
	if (typeof document === 'undefined') return

	const dialog = document.createElement('dialog')
	dialog.className =
		'modal modal-open z-60 bg-neutral/60 backdrop-blur-xs flex items-center justify-center p-2 sm:p-6'
	dialog.setAttribute('aria-label', 'Mermaid Diagram Fullscreen View')
	dialog.setAttribute('data-testid', 'mermaid-lightbox-dialog')

	dialog.innerHTML = `
		<div class="modal-box w-full max-w-[96vw] h-[92vh] max-h-[92vh] p-4 flex flex-col bg-base-100 rounded-2xl border border-base-300 shadow-2xl relative">
			<div class="flex items-center justify-between pb-2 border-b border-base-200">
				<span class="text-xs sm:text-sm font-semibold text-base-content/80">Architectural Diagram Preview</span>
				<button type="button" class="btn btn-sm btn-circle btn-ghost text-base-content/70 hover:text-base-content" data-testid="mermaid-lightbox-close" aria-label="Close fullscreen view">✕</button>
			</div>
			<div class="flex-1 overflow-auto flex items-center justify-center p-2 sm:p-4">
				<div class="w-full h-full flex items-center justify-center [&>svg]:max-w-full [&>svg]:max-h-full">
					${svgHtml}
				</div>
			</div>
		</div>
	`

	const close = () => {
		dialog.remove()
		window.removeEventListener('keydown', onKey, true)
	}

	// Capture phase + stopPropagation: Escape closes only the lightbox,
	// not the task modal underneath (which listens on window, bubble phase).
	const onKey = (e: KeyboardEvent) => {
		if (e.key === 'Escape') {
			e.stopPropagation()
			close()
		}
	}

	dialog
		.querySelector('[data-testid="mermaid-lightbox-close"]')
		?.addEventListener('click', close)
	dialog.addEventListener('click', (e) => {
		if (e.target === dialog) close()
	})
	window.addEventListener('keydown', onKey, true)
	document.body.appendChild(dialog)
}

function createDiagramElement(svg: string, rawCode: string): HTMLElement {
	const wrapper = document.createElement('div')
	wrapper.className =
		'mermaid-wrapper mermaid-diagram relative group my-4 rounded-xl border border-base-200/80 bg-base-200/30 overflow-hidden select-none'
	wrapper.setAttribute('data-testid', 'mermaid-diagram')
	wrapper.setAttribute('data-mermaid-source', rawCode)

	let currentScale = 1.0

	// Diagram action toolbar
	const toolbar = document.createElement('div')
	toolbar.className =
		'mermaid-toolbar absolute top-2 right-2 flex items-center gap-1 bg-base-100/90 backdrop-blur-xs border border-base-300/80 rounded-lg p-1 shadow-xs z-10 opacity-70 group-hover:opacity-100 transition-opacity'

	// Zoom In
	const zoomInBtn = document.createElement('button')
	zoomInBtn.type = 'button'
	zoomInBtn.className = 'btn btn-xs btn-ghost btn-square'
	zoomInBtn.title = 'Zoom In'
	zoomInBtn.setAttribute('aria-label', 'Zoom in diagram')
	zoomInBtn.setAttribute('data-testid', 'mermaid-zoom-in')
	zoomInBtn.innerHTML =
		'<svg class="size-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4"/></svg>'

	// Zoom Out
	const zoomOutBtn = document.createElement('button')
	zoomOutBtn.type = 'button'
	zoomOutBtn.className = 'btn btn-xs btn-ghost btn-square'
	zoomOutBtn.title = 'Zoom Out'
	zoomOutBtn.setAttribute('aria-label', 'Zoom out diagram')
	zoomOutBtn.setAttribute('data-testid', 'mermaid-zoom-out')
	zoomOutBtn.innerHTML =
		'<svg class="size-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M20 12H4"/></svg>'

	// Zoom Reset
	const zoomResetBtn = document.createElement('button')
	zoomResetBtn.type = 'button'
	zoomResetBtn.className = 'btn btn-xs btn-ghost px-1.5 text-[11px] font-mono'
	zoomResetBtn.title = 'Reset Zoom'
	zoomResetBtn.setAttribute('aria-label', 'Reset diagram zoom')
	zoomResetBtn.setAttribute('data-testid', 'mermaid-zoom-reset')
	zoomResetBtn.textContent = '100%'

	// Expand / Fullscreen
	const expandBtn = document.createElement('button')
	expandBtn.type = 'button'
	expandBtn.className = 'btn btn-xs btn-ghost btn-square'
	expandBtn.title = 'Fullscreen Preview'
	expandBtn.setAttribute('aria-label', 'Expand diagram to fullscreen')
	expandBtn.setAttribute('data-testid', 'mermaid-expand-btn')
	expandBtn.innerHTML =
		'<svg class="size-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M4 8V4m0 0h4M4 4l5 5m11-5h-4m4 0v4m0 0l-5-5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4"/></svg>'

	toolbar.appendChild(zoomInBtn)
	toolbar.appendChild(zoomOutBtn)
	toolbar.appendChild(zoomResetBtn)
	toolbar.appendChild(expandBtn)

	// Viewport & Inner container
	const viewport = document.createElement('div')
	viewport.className =
		'mermaid-viewport overflow-auto p-4 flex justify-center min-h-[160px] max-h-[550px]'

	const inner = document.createElement('div')
	inner.className =
		'mermaid-inner transition-transform duration-150 origin-top flex justify-center w-full'
	inner.innerHTML = svg

	viewport.appendChild(inner)
	wrapper.appendChild(toolbar)
	wrapper.appendChild(viewport)

	const updateZoom = (newScale: number) => {
		currentScale = Math.min(Math.max(newScale, 0.4), 3.0)
		inner.style.transform = currentScale === 1.0 ? '' : `scale(${currentScale})`
		zoomResetBtn.textContent = `${Math.round(currentScale * 100)}%`
	}

	zoomInBtn.addEventListener('click', () => updateZoom(currentScale + 0.25))
	zoomOutBtn.addEventListener('click', () => updateZoom(currentScale - 0.25))
	zoomResetBtn.addEventListener('click', () => updateZoom(1.0))
	expandBtn.addEventListener('click', () => {
		const currentSvg = inner.innerHTML
		openDiagramLightbox(currentSvg)
	})

	return wrapper
}

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

		if (preEl.getAttribute('data-mermaid-rendering') === 'true') continue
		preEl.setAttribute('data-mermaid-rendering', 'true')

		// Store raw source code on the pre element so theme switches can re-render it
		const rawCode =
			preEl.getAttribute('data-mermaid-source') || codeEl.textContent || ''
		if (!rawCode.trim()) continue

		preEl.setAttribute('data-mermaid-source', rawCode)

		const diagramId = `mermaid-svg-${Date.now()}-${++diagramSeq}`
		try {
			const { svg } = await mermaid.render(diagramId, rawCode)

			const wrapper = createDiagramElement(svg, rawCode)
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

	const rendered = container.querySelectorAll<HTMLElement>(
		'.mermaid-diagram[data-mermaid-source]',
	)
	if (rendered.length === 0) {
		await renderMermaidDiagrams(container)
		return
	}

	// Force re-config with new theme
	currentTheme = null
	configureMermaid()

	for (const el of Array.from(rendered)) {
		const rawCode = el.getAttribute('data-mermaid-source') || ''
		if (!rawCode.trim()) continue

		const diagramId = `mermaid-svg-${Date.now()}-${++diagramSeq}`
		try {
			const { svg } = await mermaid.render(diagramId, rawCode)
			const inner = el.querySelector<HTMLElement>('.mermaid-inner')
			if (inner) {
				inner.innerHTML = svg
			} else {
				el.innerHTML = svg
			}
		} catch (err) {
			console.warn(
				'[Mermaid] Failed to re-render diagram on theme change:',
				err,
			)
		}
	}
}
