import { render } from 'preact'
import { App } from './App'
import { initTheme } from './state/theme'
import { initDocumentLang } from './utils/i18n'
import './styles.css'

if (process.env.NODE_ENV !== 'production') {
	await import('preact/debug')
}

// Initialize theme before mounting
initTheme()
initDocumentLang()

const root = document.getElementById('app')
if (root) {
	render(<App />, root)
}
