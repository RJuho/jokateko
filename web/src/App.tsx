import { useEffect } from 'preact/hooks'
import { KanbanBoard } from './components/board/KanbanBoard'
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
import { bootstrap } from './state/bootstrap'
import { startSSE, stopSSE } from './state/sse'
import {
	activeTab,
	fetchLiveBoard,
	fetchLiveEntities,
	initFromSnapshot,
	mode,
	tasks,
} from './state/store'

export function App() {
	const currentTab = activeTab.value

	useEffect(() => {
		const res = bootstrap()
		if (res.mode === 'static' && res.snapshot) {
			initFromSnapshot(res.snapshot, res.warnings)
		} else {
			mode.value = 'live'
			fetchLiveBoard().then((success) => {
				if (!success && tasks.value.length === 0) {
					initFromSnapshot(sampleSnapshot)
				}
			})
			fetchLiveEntities()
			startSSE()
		}

		return () => {
			stopSSE()
		}
	}, [])

	return (
		<div class='min-h-screen bg-base-100 flex flex-col font-sans text-base-content selection:bg-primary/20 selection:text-primary'>
			{/* Application Header */}
			<Header />

			{/* Main Content Area */}
			<div class='flex-1 flex flex-col'>
				<div class='max-w-7xl w-full mx-auto px-4 sm:px-6 pt-3'>
					<ValidationBanner />
				</div>

				{currentTab === 'board' && (
					<>
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
