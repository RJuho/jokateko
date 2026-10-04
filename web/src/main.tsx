import { render } from 'preact'
import { App } from './App'
import { initTheme } from './state/theme'
import './styles.css'

if (process.env.NODE_ENV !== 'production') {
	await import('preact/debug')
}

// Initialize theme before mounting
initTheme()

const root = document.getElementById('app')
if (root) {
	render(<App />, root)
}
