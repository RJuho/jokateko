import logoSvg from '../logo.svg' with { type: 'text' }

// Inlined as a data URI: one source file, no duplicate gradient IDs in the DOM,
// and the CSP already allows img-src data:
export const logoSrc = `data:image/svg+xml,${encodeURIComponent(logoSvg)}`
