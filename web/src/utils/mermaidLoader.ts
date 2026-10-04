import type { Mermaid } from 'mermaid'

// Injected by scripts/bundle.ts from the lockfile-pinned node_modules/mermaid
// (absent in the dev server, where the asset is served without integrity).
declare const __MERMAID_VERSION__: string
declare const __MERMAID_INTEGRITY__: string

const MERMAID_VERSION =
	typeof __MERMAID_VERSION__ === 'string' ? __MERMAID_VERSION__ : 'dev'
const MERMAID_INTEGRITY =
	typeof __MERMAID_INTEGRITY__ === 'string' ? __MERMAID_INTEGRITY__ : ''

/**
 * Where the Mermaid runtime comes from:
 * - `asset`   (default, live daemon): versioned file embedded in the binary
 * - `bundled` (static export): inert inline block executed on first use
 * - `cdn`     (static export): jsDelivr, same version, SRI-verified
 * - `none`    (static export): diagrams stay as code blocks
 * The static exporter selects the mode with <meta name="jokateko-mermaid">.
 */
export type MermaidSource = 'asset' | 'bundled' | 'cdn' | 'none'

/** Same-origin path of the embedded asset, versioned so it can be cached immutably. */
export const MERMAID_ASSET_PATH = `/assets/mermaid-${MERMAID_VERSION}.min.js`
export const MERMAID_CDN_URL = `https://cdn.jsdelivr.net/npm/mermaid@${MERMAID_VERSION}/dist/mermaid.min.js`

export function mermaidSource(): MermaidSource {
	const mode = document
		.querySelector<HTMLMetaElement>('meta[name="jokateko-mermaid"]')
		?.content.trim()
	if (mode === 'bundled' || mode === 'cdn' || mode === 'none') {
		return mode
	}
	return 'asset'
}

function globalMermaid(): Mermaid | null {
	return (globalThis as { mermaid?: Mermaid }).mermaid ?? null
}

function loadScript(src: string): Promise<void> {
	return new Promise((resolve, reject) => {
		const script = document.createElement('script')
		script.src = src
		script.async = true
		if (MERMAID_INTEGRITY) {
			script.integrity = MERMAID_INTEGRITY
			script.crossOrigin = 'anonymous'
		}
		script.onload = () => resolve()
		script.onerror = () => reject(new Error(`failed to load ${src}`))
		document.head.appendChild(script)
	})
}

async function load(): Promise<Mermaid | null> {
	const existing = globalMermaid()
	if (existing) return existing

	switch (mermaidSource()) {
		case 'none':
			return null
		case 'bundled': {
			// The inert block follows the app script; on a deep link a diagram can be
			// requested while the parser is still streaming it, so wait for it to finish.
			if (document.readyState === 'loading') {
				await new Promise((resolve) =>
					document.addEventListener('DOMContentLoaded', resolve, {
						once: true,
					}),
				)
			}
			const block = document.getElementById('jokateko-mermaid-src')
			if (!block?.textContent) {
				throw new Error('bundled Mermaid block is missing')
			}
			const script = document.createElement('script')
			script.textContent = block.textContent
			document.head.appendChild(script)
			break
		}
		case 'cdn':
			await loadScript(MERMAID_CDN_URL)
			break
		default:
			await loadScript(MERMAID_ASSET_PATH)
	}
	return globalMermaid()
}

let pending: Promise<Mermaid | null> | null = null

/**
 * Loads the Mermaid runtime once, on first use. Resolves to null when diagrams
 * are disabled or the runtime cannot be loaded (e.g. CDN while offline), so
 * callers keep the raw code blocks. A failed load is retried on the next call.
 */
export function loadMermaid(): Promise<Mermaid | null> {
	if (!pending) {
		pending = load().catch((err) => {
			console.warn(
				'[Mermaid] Runtime unavailable, keeping raw code blocks:',
				err,
			)
			pending = null
			return null
		})
	}
	return pending
}
