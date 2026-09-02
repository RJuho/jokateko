import { render } from 'preact'
import { App } from './App'
import './styles.css'

// Initialize theme before mounting
try {
	let theme = localStorage.getItem('theme')
	if (!theme) {
		const prefersDark =
			!window.matchMedia
			|| window.matchMedia('(prefers-color-scheme: dark)').matches
		theme = prefersDark ? 'sunset' : 'winter'
	}
	document.documentElement.setAttribute('data-theme', theme)
} catch {
	// ignore
}

const root = document.getElementById('app')
if (root) {
	render(<App />, root)
}
