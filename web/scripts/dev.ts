import index from '../index.html'

const port = Number(process.env.PORT) || 4321
const host = process.env.HOST || '0.0.0.0'
const backendUrl = process.env.BACKEND_URL || 'http://127.0.0.1:8080'

const server = Bun.serve({
	hostname: host,
	port,
	development: true,
	routes: {
		'/': index,
	},
	async fetch(req) {
		const url = new URL(req.url)

		// Reverse proxy /api/* requests to Go backend if running
		if (url.pathname.startsWith('/api')) {
			try {
				const target = new URL(url.pathname + url.search, backendUrl)
				return await fetch(target.href, {
					method: req.method,
					headers: req.headers,
					body: req.body,
				})
			} catch {
				return new Response(
					JSON.stringify({
						error:
							'Go backend is offline. Run "jokateko serve" or export BACKEND_URL.',
					}),
					{
						status: 502,
						headers: { 'Content-Type': 'application/json' },
					},
				)
			}
		}

		// Lazily loaded Mermaid runtime (the dev build has no version/SRI defines)
		if (url.pathname === '/assets/mermaid-dev.min.js') {
			return new Response(
				Bun.file(
					new URL(
						'../node_modules/mermaid/dist/mermaid.min.js',
						import.meta.url,
					),
				),
				{ headers: { 'Content-Type': 'text/javascript; charset=utf-8' } },
			)
		}

		// Fallback for client-side routing
		return new Response(index as unknown as BlobPart, {
			headers: { 'Content-Type': 'text/html; charset=utf-8' },
		})
	},
})

console.log('🚀 Jokateko UI Dev Server running with Hot Reload:')
console.log(`   ➜ Local:        http://localhost:${server.port}/`)
console.log(`   ➜ Network:      http://${server.hostname}:${server.port}/`)
if (host === '0.0.0.0') {
	console.log(`   ➜ Devcontainer: http://127.0.0.1:${server.port}/`)
}
