import { Maximize2, X, ZoomIn, ZoomOut } from 'lucide-preact'
import type { Mermaid } from 'mermaid'
import { h, render } from 'preact'
import { isDarkTheme } from '../state/theme'
import { attachDiagramZoom, type FitMode } from './diagramViewport'
import { loadMermaid } from './mermaidLoader'

let mermaidInitialized = false
let currentTheme: 'dark' | 'default' | null = null

function configureMermaid(mermaid: Mermaid, forceTheme?: 'dark' | 'default') {
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

type IconComponent = typeof ZoomIn

function iconButton(
	icon: IconComponent,
	label: string,
	testId: string,
	onClick: () => void,
): HTMLButtonElement {
	const btn = document.createElement('button')
	btn.type = 'button'
	btn.className = 'btn btn-xs btn-ghost btn-square'
	btn.title = label
	btn.setAttribute('aria-label', label)
	btn.setAttribute('data-testid', testId)
	render(h(icon, { class: 'size-3.5' }), btn)
	btn.addEventListener('click', onClick)
	return btn
}

/**
 * Builds a pan/zoom diagram viewport plus its toolbar (zoom in/out, reset
 * readout, and an optional trailing action such as fullscreen or close).
 */
function createZoomableDiagram(
	svgHtml: string,
	fit: FitMode,
	viewportClass: string,
	trailing?: HTMLButtonElement,
) {
	const viewport = document.createElement('div')
	viewport.className = `mermaid-viewport overflow-auto cursor-grab flex justify-center-safe items-start focus-visible:outline-2 focus-visible:outline-primary ${viewportClass}`
	viewport.tabIndex = 0
	viewport.setAttribute('role', 'group')
	viewport.setAttribute(
		'aria-label',
		'Diagram: drag or scroll to pan, Ctrl + wheel or + / - to zoom, 0 to reset',
	)

	const inner = document.createElement('div')
	inner.className = 'mermaid-inner shrink-0'
	inner.innerHTML = svgHtml
	viewport.appendChild(inner)

	const zoomResetBtn = document.createElement('button')
	zoomResetBtn.type = 'button'
	zoomResetBtn.className = 'btn btn-xs btn-ghost px-1.5 text-[11px] font-mono'
	zoomResetBtn.title = 'Reset Zoom'
	zoomResetBtn.setAttribute('aria-label', 'Reset diagram zoom')
	zoomResetBtn.setAttribute('data-testid', 'mermaid-zoom-reset')
	zoomResetBtn.textContent = '100%'

	const zoom = attachDiagramZoom(viewport, inner, fit, (scale) => {
		zoomResetBtn.textContent = `${Math.round(scale * 100)}%`
	})
	zoomResetBtn.addEventListener('click', () => zoom.reset())

	const toolbar = document.createElement('div')
	toolbar.className =
		'mermaid-toolbar flex items-center gap-1 bg-base-100/90 backdrop-blur-xs border border-base-300/80 rounded-lg p-1 shadow-xs'
	toolbar.append(
		iconButton(ZoomIn, 'Zoom in diagram', 'mermaid-zoom-in', () =>
			zoom.zoomBy(1.25),
		),
		iconButton(ZoomOut, 'Zoom out diagram', 'mermaid-zoom-out', () =>
			zoom.zoomBy(1 / 1.25),
		),
		zoomResetBtn,
	)
	if (trailing) toolbar.append(trailing)

	return { viewport, inner, toolbar, zoom }
}

export function openDiagramLightbox(svgHtml: string) {
	if (typeof document === 'undefined') return

	const dialog = document.createElement('dialog')
	dialog.className =
		'modal modal-open z-60 bg-neutral/60 backdrop-blur-xs flex items-center justify-center p-2 sm:p-6'
	dialog.setAttribute('aria-label', 'Mermaid Diagram Fullscreen View')
	dialog.setAttribute('data-testid', 'mermaid-lightbox-dialog')

	const box = document.createElement('div')
	box.className =
		'modal-box w-full max-w-[96vw] h-[92vh] max-h-[92vh] p-4 flex flex-col bg-base-100 rounded-2xl border border-base-300 shadow-2xl relative select-none'

	const closeBtn = iconButton(
		X,
		'Close fullscreen view',
		'mermaid-lightbox-close',
		() => close(),
	)
	const diagram = createZoomableDiagram(
		svgHtml,
		'contain',
		'flex-1 min-h-0 p-2 sm:p-4',
		closeBtn,
	)

	const header = document.createElement('div')
	header.className =
		'flex items-center justify-between gap-2 pb-2 border-b border-base-200'
	const title = document.createElement('span')
	title.className = 'text-xs sm:text-sm font-semibold text-base-content/80'
	title.textContent = 'Architectural Diagram Preview'
	header.append(title, diagram.toolbar)

	box.append(header, diagram.viewport)
	dialog.append(box)

	const close = () => {
		diagram.zoom.destroy()
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

	dialog.addEventListener('click', (e) => {
		if (e.target === dialog) close()
	})
	window.addEventListener('keydown', onKey, true)
	document.body.appendChild(dialog)
	diagram.viewport.focus()
}

function createDiagramElement(svg: string, rawCode: string): HTMLElement {
	const wrapper = document.createElement('div')
	wrapper.className =
		'mermaid-wrapper mermaid-diagram relative group my-4 rounded-xl border border-base-200/80 bg-base-200/30 overflow-hidden select-none'
	wrapper.setAttribute('data-testid', 'mermaid-diagram')
	wrapper.setAttribute('data-mermaid-source', rawCode)

	const inner: { el?: HTMLElement } = {}
	const expandBtn = iconButton(
		Maximize2,
		'Expand diagram to fullscreen',
		'mermaid-expand-btn',
		() => openDiagramLightbox(inner.el?.innerHTML ?? ''),
	)
	const diagram = createZoomableDiagram(
		svg,
		'width',
		'p-4 min-h-[160px] max-h-[550px]',
		expandBtn,
	)
	inner.el = diagram.inner
	diagram.toolbar.classList.add(
		'absolute',
		'top-2',
		'right-2',
		'z-10',
		'opacity-70',
		'group-hover:opacity-100',
		'focus-within:opacity-100',
		'transition-opacity',
	)

	wrapper.append(diagram.toolbar, diagram.viewport)
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

	// Runtime is loaded lazily, only once a diagram is actually on screen
	const mermaid = await loadMermaid()
	if (!mermaid) return
	configureMermaid(mermaid)

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

	const mermaid = await loadMermaid()
	if (!mermaid) return

	// Force re-config with new theme
	currentTheme = null
	configureMermaid(mermaid)

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
