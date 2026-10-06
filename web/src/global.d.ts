declare module '*.css' {
	const content: string
	export default content
}

interface HTMLBundle {
	[key: string]: unknown
}

declare module '*.html' {
	const content: HTMLBundle
	export default content
}

declare module '*.svg' {
	const content: string
	export default content
}
