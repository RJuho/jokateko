import type { RefObject } from 'preact'
import { useEffect, useRef } from 'preact/hooks'
import { isDarkTheme } from '../state/theme'
import {
	renderMermaidDiagrams,
	reRenderMermaidDiagrams,
} from '../utils/mermaid'

/**
 * Renders Mermaid code blocks inside `ref` whenever `deps` change, and
 * re-renders existing diagrams when the light/dark theme flips.
 */
export function useMermaidDiagrams(
	ref: RefObject<HTMLElement | null>,
	deps: unknown[],
): void {
	const isDark = isDarkTheme.value
	const renderedTheme = useRef(isDark)

	useEffect(() => {
		if (ref.current) {
			renderMermaidDiagrams(ref.current)
		}
	}, deps)

	useEffect(() => {
		if (renderedTheme.current === isDark) {
			return
		}
		renderedTheme.current = isDark
		if (ref.current) {
			reRenderMermaidDiagrams(ref.current)
		}
	}, [isDark])
}
