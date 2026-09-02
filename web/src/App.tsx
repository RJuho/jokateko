import { useEffect } from 'preact/hooks'
import { KanbanBoard } from './components/board/KanbanBoard'
import { MilestoneCards } from './components/board/MilestoneCards'
import { FilterBar } from './components/common/FilterBar'
import { Header } from './components/common/Header'
import { ValidationBanner } from './components/common/ValidationBanner'
import { GlossaryView } from './components/glossary/GlossaryView'
import { MilestonesView } from './components/milestones/MilestonesView'
import { CreateTaskModal } from './components/modal/CreateTaskModal'
import { TaskDetailModal } from './components/modal/TaskDetailModal'
import { TaskEditModal } from './components/modal/TaskEditModal'
import { StrategiesView } from './components/strategies/StrategiesView'
import { sampleSnapshot } from './fixtures/sampleData'
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
			} else {
				initFromSnapshot(sampleSnapshot, [], 'client')
			}
		} else if (res.mode === 'static' && res.snapshot) {
			initFromSnapshot(res.snapshot, res.warnings, 'static')
		} else if (res.mode === 'live') {
			mode.value = 'live'
			fetchLiveBoard().then((success) => {
				if (!success && tasks.value.length === 0) {
					initFromSnapshot(sampleSnapshot, [], 'client')
				}
			})
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
		<div class='min-h-screen bg-base-100 flex flex-col font-sans text-base-content selection:bg-primary/20 selection:text-primary'>
			{/* Application Header */}
			<Header />

			{/* Main Content Area */}
			<div class='flex-1 flex flex-col'>
				<div class='w-full px-3 sm:px-4 md:px-6 pt-2'>
					<ValidationBanner />
				</div>

				{currentTab === 'board' && (
					<>
						<MilestoneCards />
						<FilterBar />
						<KanbanBoard />
					</>
				)}

				{currentTab === 'milestones' && <MilestonesView />}

				{currentTab === 'strategies' && <StrategiesView />}

				{currentTab === 'glossary' && <GlossaryView />}
			</div>

			{/* Interactive Modals */}
			<TaskDetailModal />
			<TaskEditModal />
			<CreateTaskModal />
		</div>
	)
}
