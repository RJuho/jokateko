import { useEffect } from 'preact/hooks'
import { KanbanBoard } from './components/board/KanbanBoard'
import { MilestoneCards } from './components/board/MilestoneCards'
import { FilterBar } from './components/common/FilterBar'
import { Footer } from './components/common/Footer'
import { Header } from './components/common/Header'
import { ValidationBanner } from './components/common/ValidationBanner'
import { GlossaryView } from './components/glossary/GlossaryView'
import { MilestonesView } from './components/milestones/MilestonesView'
import { CreateTaskModal } from './components/modal/CreateTaskModal'
import { TaskDetailModal } from './components/modal/TaskDetailModal'
import { TaskEditModal } from './components/modal/TaskEditModal'
import { StrategiesView } from './components/strategies/StrategiesView'
import { initRouter, updateDocumentTitle } from './router'
import { bootstrap } from './state/bootstrap'
import { startSSE, stopSSE } from './state/sse'
import {
	activeGlossaryId,
	activeStrategyId,
	activeTab,
	activeTaskDetailId,
	config,
	fetchLiveBoard,
	fetchLiveEntities,
	filters,
	glossary,
	initFromSnapshot,
	milestones,
	mode,
	strategies,
	tasks,
} from './state/store'

export function App() {
	const currentTab = activeTab.value

	useEffect(() => {
		const cleanupRouter = initRouter()
		const res = bootstrap()
		if (res.mode === 'client') {
			if (res.snapshot) {
				initFromSnapshot(res.snapshot, res.warnings, 'client')
			}
		} else if (res.mode === 'static' && res.snapshot) {
			initFromSnapshot(res.snapshot, res.warnings, 'static')
		} else if (res.mode === 'live') {
			mode.value = 'live'
			fetchLiveBoard()
			fetchLiveEntities()
			startSSE()
		}

		return () => {
			cleanupRouter()
			stopSSE()
		}
	}, [])

	useEffect(() => {
		updateDocumentTitle()
	}, [
		activeTab.value,
		activeTaskDetailId.value,
		activeStrategyId.value,
		activeGlossaryId.value,
		filters.value.selectedMilestone,
		tasks.value,
		milestones.value,
		strategies.value,
		glossary.value,
		config.value,
	])

	return (
		<div class='h-dvh bg-base-100 flex flex-col font-sans text-base-content overflow-hidden'>
			{/* Application Header */}
			<Header />

			{/* Main Content Area: Fits fixed screen height without double body scrollbars */}
			<div class='flex-1 min-h-0 flex flex-col overflow-hidden'>
				<div class='w-full px-3 sm:px-4 md:px-6 pt-2 shrink-0'>
					<ValidationBanner />
				</div>

				{currentTab === 'board' && (
					<div class='flex-1 min-h-0 flex flex-col overflow-hidden'>
						<div class='shrink-0'>
							<MilestoneCards />
						</div>
						<div class='shrink-0'>
							<FilterBar />
						</div>
						<KanbanBoard />
					</div>
				)}

				{currentTab === 'milestones' && (
					<div class='flex-1 min-h-0 overflow-y-auto'>
						<MilestonesView />
					</div>
				)}

				{currentTab === 'strategies' && (
					<div class='flex-1 min-h-0 overflow-y-auto'>
						<StrategiesView />
					</div>
				)}

				{currentTab === 'glossary' && (
					<div class='flex-1 min-h-0 overflow-y-auto'>
						<GlossaryView />
					</div>
				)}
			</div>

			{/* Minimalistic Footer on all pages (desktop bottom bar, hidden on mobile) */}
			<Footer />

			{/* Interactive Modals */}
			<TaskDetailModal />
			<TaskEditModal />
			<CreateTaskModal />
		</div>
	)
}
