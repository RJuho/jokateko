/**
 * Pan & zoom for rendered Mermaid SVGs, using the browser's own scrolling:
 * - zoom resizes the SVG's layout box (not a CSS transform), so the scroll area
 *   grows and every edge of a zoomed diagram stays reachable
 * - mouse/pen drag pans; plain wheel and touch scroll natively
 * - Ctrl/⌘ + wheel (and trackpad pinch) zooms around the cursor
 * - keyboard (viewport is focusable): arrows scroll natively, + / - / 0 zoom
 * 100% means "fit": the card fits the width without upscaling, the lightbox
 * fits (contains) the whole diagram.
 */

export type FitMode = 'width' | 'contain'

export interface DiagramZoom {
	zoomBy: (factor: number, clientX?: number, clientY?: number) => void
	reset: () => void
	destroy: () => void
}

const MIN_SCALE = 0.25
const MAX_SCALE = 5

function naturalSize(svg: SVGSVGElement): { width: number; height: number } {
	const vb = svg.viewBox?.baseVal
	if (vb && vb.width > 0 && vb.height > 0) {
		return { width: vb.width, height: vb.height }
	}
	const rect = svg.getBoundingClientRect()
	return { width: rect.width || 1, height: rect.height || 1 }
}

export function attachDiagramZoom(
	viewport: HTMLElement,
	inner: HTMLElement,
	fit: FitMode,
	onScaleChange: (scale: number) => void,
): DiagramZoom {
	let scale = 1

	const apply = () => {
		const svg = inner.querySelector('svg')
		if (!svg) return
		const nat = naturalSize(svg)
		const style = getComputedStyle(viewport)
		const availW =
			viewport.clientWidth
			- Number.parseFloat(style.paddingLeft)
			- Number.parseFloat(style.paddingRight)
		const availH =
			viewport.clientHeight
			- Number.parseFloat(style.paddingTop)
			- Number.parseFloat(style.paddingBottom)
		const fitScale =
			fit === 'contain'
				? Math.min(availW / nat.width, availH / nat.height)
				: Math.min(1, availW / nat.width)
		const px = Math.max(fitScale, 0.01) * scale
		svg.style.maxWidth = 'none'
		svg.style.maxHeight = 'none'
		svg.style.width = `${nat.width * px}px`
		svg.style.height = `${nat.height * px}px`
	}

	const setScale = (next: number, clientX?: number, clientY?: number) => {
		const clamped = Math.min(MAX_SCALE, Math.max(MIN_SCALE, next))
		if (clamped === scale) return
		const rect = viewport.getBoundingClientRect()
		// Keep the point under the cursor (or the viewport centre) in place
		const ox = (clientX ?? rect.left + rect.width / 2) - rect.left
		const oy = (clientY ?? rect.top + rect.height / 2) - rect.top
		const contentX = viewport.scrollLeft + ox
		const contentY = viewport.scrollTop + oy
		const ratio = clamped / scale
		scale = clamped
		apply()
		viewport.scrollLeft = contentX * ratio - ox
		viewport.scrollTop = contentY * ratio - oy
		onScaleChange(scale)
	}

	// Drag to pan (mouse / pen; touch keeps native scrolling)
	let drag: { x: number; y: number; left: number; top: number } | null = null
	const onPointerDown = (e: PointerEvent) => {
		if (e.pointerType === 'touch' || e.button !== 0) return
		drag = {
			x: e.clientX,
			y: e.clientY,
			left: viewport.scrollLeft,
			top: viewport.scrollTop,
		}
		viewport.setPointerCapture(e.pointerId)
		viewport.classList.add('cursor-grabbing')
		e.preventDefault()
	}
	const onPointerMove = (e: PointerEvent) => {
		if (!drag) return
		viewport.scrollLeft = drag.left - (e.clientX - drag.x)
		viewport.scrollTop = drag.top - (e.clientY - drag.y)
	}
	const endDrag = (e: PointerEvent) => {
		if (!drag) return
		drag = null
		if (viewport.hasPointerCapture(e.pointerId)) {
			viewport.releasePointerCapture(e.pointerId)
		}
		viewport.classList.remove('cursor-grabbing')
	}

	// Ctrl/⌘ + wheel (trackpad pinch reports ctrlKey) zooms; plain wheel scrolls natively
	const onWheel = (e: WheelEvent) => {
		if (!e.ctrlKey && !e.metaKey) return
		e.preventDefault()
		setScale(scale * Math.exp(-e.deltaY * 0.002), e.clientX, e.clientY)
	}

	const onKeyDown = (e: KeyboardEvent) => {
		if (e.ctrlKey || e.metaKey || e.altKey) return
		if (e.key === '+' || e.key === '=') setScale(scale * 1.25)
		else if (e.key === '-' || e.key === '_') setScale(scale / 1.25)
		else if (e.key === '0') reset()
		else return
		e.preventDefault()
	}

	const reset = () => {
		scale = 1
		apply()
		viewport.scrollLeft = 0
		viewport.scrollTop = 0
		onScaleChange(scale)
	}

	viewport.addEventListener('pointerdown', onPointerDown)
	viewport.addEventListener('pointermove', onPointerMove)
	viewport.addEventListener('pointerup', endDrag)
	viewport.addEventListener('pointercancel', endDrag)
	viewport.addEventListener('wheel', onWheel, { passive: false })
	viewport.addEventListener('keydown', onKeyDown)

	// Re-fit when the viewport resizes or the SVG is replaced (theme re-render)
	const resizeObserver = new ResizeObserver(() => apply())
	resizeObserver.observe(viewport)
	const mutationObserver = new MutationObserver(() => apply())
	mutationObserver.observe(inner, { childList: true })
	apply()

	return {
		zoomBy: (factor, clientX, clientY) =>
			setScale(scale * factor, clientX, clientY),
		reset,
		destroy: () => {
			resizeObserver.disconnect()
			mutationObserver.disconnect()
			viewport.removeEventListener('pointerdown', onPointerDown)
			viewport.removeEventListener('pointermove', onPointerMove)
			viewport.removeEventListener('pointerup', endDrag)
			viewport.removeEventListener('pointercancel', endDrag)
			viewport.removeEventListener('wheel', onWheel)
			viewport.removeEventListener('keydown', onKeyDown)
		},
	}
}
