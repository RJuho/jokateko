import type { Snapshot } from '../schemas/models'

export const sampleSnapshot: Snapshot = {
	config: {
		project: {
			name: 'Jokateko Kanban',
			description: 'Professional Tasks-as-Code Kanban System',
		},
		board: {
			columns: [
				{ id: 'backlog', name: 'Backlog', color: '#94a3b8' },
				{ id: 'ready', name: 'Ready', color: '#38bdf8' },
				{ id: 'in_progress', name: 'In Progress', color: '#fbbf24' },
				{ id: 'in_review', name: 'In Review', color: '#c084fc' },
				{ id: 'done', name: 'Done', color: '#4ade80' },
			],
		},
		tags: {
			allowed: [
				'frontend',
				'backend',
				'database',
				'security',
				'ui/ux',
				'performance',
			],
			enforce_allowed: false,
		},
		build: {
			time: '2026-09-02T05:30:00Z',
			branch: 'main',
			commit: '03c4a07',
		},
	},
	tasks: [
		{
			id: '260901-user-auth',
			title: 'Implement OAuth2 and Session Authentication',
			status: 'in_progress',
			priority: 'critical',
			milestone: 'm1-mvp-release',
			tags: ['security', 'backend'],
			summary:
				'Secure login flow using modern OAuth2 and HTTP-only session cookies.',
			dependencies: [],
			body: `## Acceptance Criteria
- [x] Configure authorization endpoint
- [x] Validate PKCE challenge
- [ ] Implement refresh token rotation
- [ ] Write integration tests for expired tokens

## Implementation Notes
Use strict SameSite cookies and ensure TLS termination is respected.`,
			total_criteria: 4,
			completed_criteria: 2,
		},
		{
			id: '260901-kanban-board',
			title: 'Design Mobile-First Responsive Kanban Board',
			status: 'in_progress',
			priority: 'high',
			milestone: 'm1-mvp-release',
			tags: ['frontend', 'ui/ux'],
			summary:
				'Mobile-first Kanban columns with daisyUI styling and smooth touch navigation.',
			dependencies: [],
			body: `## Acceptance Criteria
- [x] Responsive layout with touch-friendly columns
- [x] Accessible card details modal with keyboard navigation
- [ ] Drag-and-drop column transitions
- [ ] Real-time SSE updates reflection

## Design Requirements
Ensure full compliance with daisyUI 5 and WCAG AAA color contrast.`,
			total_criteria: 4,
			completed_criteria: 2,
		},
		{
			id: '260901-sqlite-fts5',
			title: 'Optimize SQLite FTS5 Search Indexing',
			status: 'ready',
			priority: 'medium',
			milestone: 'm2-performance',
			tags: ['database', 'performance'],
			summary:
				'Full-text search queries optimization across tasks, strategies, and glossary.',
			dependencies: [],
			body: `## Acceptance Criteria
- [ ] Implement porter stemmer tokenizer
- [ ] Measure BM25 score ranking
- [ ] Ensure sub-5ms search latency across 10,000 tasks`,
			total_criteria: 3,
			completed_criteria: 0,
		},
		{
			id: '260901-ci-cd-pipeline',
			title: 'Configure Zero-Dependency CI/CD Pipeline',
			status: 'ready',
			priority: 'low',
			tags: ['backend', 'performance'],
			summary:
				'Automated GitHub Actions build matrix for Linux, macOS, and Windows binaries.',
			dependencies: [],
			body: 'Build multi-arch binaries with link-time version flags.',
			total_criteria: 0,
			completed_criteria: 0,
		},
		{
			id: '260901-blocked-deploy',
			title: 'Deploy Production Edge Cluster',
			status: 'backlog',
			priority: 'high',
			milestone: 'm1-mvp-release',
			tags: ['backend', 'security'],
			summary:
				'Deploy edge servers across 3 global regions with automated health probes.',
			dependencies: ['260901-user-auth'],
			body: 'Cannot deploy until auth and session handling are fully completed.',
			total_criteria: 2,
			completed_criteria: 0,
		},
		{
			id: '260901-spec-audit',
			title: 'Architecture & Security Specification Audit',
			status: 'done',
			priority: 'critical',
			milestone: 'm1-mvp-release',
			tags: ['security'],
			summary:
				'Completed comprehensive audit of pure Go constraints and zero CGO rules.',
			dependencies: [],
			body: `## Completion Summary
- Checked all dependencies: 100% pure Go verified.
- Confirmed SQLite modernc engine compatibility.`,
			total_criteria: 2,
			completed_criteria: 2,
		},
	],
	milestones: [
		{
			id: 'm1-mvp-release',
			title: 'v1.0 MVP Launch',
			status: 'open',
			is_archived: false,
			target_date: '2026-10-15',
			tags: ['frontend', 'backend', 'security'],
			summary: 'Initial production-ready release of Jokateko Kanban.',
			body: 'Focus on core spec adherence, fast mobile UI, and Model Context Protocol support.',
			total_tasks: 4,
			completed_tasks: 1,
			progress_percentage: 25,
		},
		{
			id: 'm2-performance',
			title: 'v1.1 Performance & Analytics',
			status: 'open',
			is_archived: false,
			target_date: '2026-12-01',
			tags: ['performance', 'database'],
			summary: 'Deep query optimization and comprehensive board metrics.',
			body: 'FTS5 benchmarking and snapshot export performance tuning.',
			total_tasks: 1,
			completed_tasks: 0,
			progress_percentage: 0,
		},
	],
	strategies: [
		{
			id: 'strat-progressive-disclosure',
			title: 'Tiered Progressive Disclosure',
			tier: 1,
			tags: ['ui/ux', 'performance'],
			summary:
				'Present high-level architectural rules first, exposing detailed criteria only on demand.',
			body: `### Core Rule
AI agents and developers should never be overwhelmed with exhaustive documentation in one shot. Provide compact summary cards, with expandable deep-dive sections.`,
		},
		{
			id: 'strat-zero-cgo',
			title: 'Zero CGO Pure Go Runtime',
			tier: 1,
			tags: ['backend', 'security'],
			summary:
				'Strict enforcement of pure Go dependencies to guarantee single-binary cross-compilation.',
			body: `### Rules
- CGO_ENABLED=0 must always compile successfully.
- Use modernc.org/sqlite for SQLite without C compiler dependency.`,
		},
		{
			id: 'strat-mobile-first-layout',
			title: 'Mobile-First Accessibility & Touch UI',
			tier: 2,
			tags: ['frontend', 'ui/ux'],
			summary:
				'All screens must adapt fluidly from 360px mobile viewports up to 4K displays.',
			body: `### Mobile Guidelines
- Minimum touch target: 44x44px.
- Use semantic ARIA attributes on all interactive elements.
- Horizontal scroll with smooth snapping for Kanban columns on small screens.`,
		},
	],
	glossary: [
		{
			id: 'term-tasks-as-code',
			title: 'Tasks-as-Code',
			tags: ['backend', 'database'],
			summary:
				'Methodology where Markdown files in the Git repository are the single source of truth for work items.',
			body: 'Tasks are represented by markdown files with TOML frontmatter stored in the repository.',
		},
		{
			id: 'term-mcp',
			title: 'Model Context Protocol (MCP)',
			tags: ['backend', 'security'],
			summary:
				'Standardized protocol enabling AI models to interact with local tools, resources, and tasks safely.',
			body: 'Jokateko acts as an MCP server allowing AI coding agents to read and manage Kanban tickets.',
		},
		{
			id: 'term-progressive-disclosure',
			title: 'Progressive Disclosure',
			tags: ['ui/ux'],
			summary:
				'An interaction design pattern that sequences information and actions across several steps to avoid cognitive overload.',
			body: 'Used in Jokateko strategies (Tier 1 -> Tier 2 -> Tier 3) and task views.',
		},
	],
}
