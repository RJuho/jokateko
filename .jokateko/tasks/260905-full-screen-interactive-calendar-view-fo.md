+++
title = 'Full-Screen Interactive Calendar View for Milestones and Tasks'
status = 'backlog'
priority = 'medium'
tags = ['feature', 'frontend']
summary = 'Build a full-screen calendar view (#calendar) displaying multi-day milestone timeline bars and single-day tasks placed by target_at or changed_at with state dots and opacity differentiation.'
dependencies = ['260904-task-and-milestone-timestamps-createdat-']
created_at = '2026-09-05T17:02:34Z'
changed_at = '2026-09-05T17:10:26Z'
+++

# Full-Screen Interactive Calendar View for Milestones and Tasks

Build a dedicated, full-screen Calendar page displaying multi-day Milestone timelines alongside single-day Tasks scheduled by target deadlines and completion timestamps.

## Background & Rationale
While Kanban boards present workflow stage columns and Milestones aggregate deliverables, teams need a chronological, calendar-based perspective to visualize deadlines, active milestones spanning several days, and delivery cadence. With standardized `created_at`, `changed_at`, and `target_at` timestamps now present, Jokateko can render an interactive timeline overview directly within the self-contained Web UI.

## Requirements & Scope
1. **Dedicated Navigation & Routing**:
   - Add a "Calendar" tab to the application header (`Header.tsx`) and mobile drawer navigation.
   - Register route `#calendar` in `router.ts` mapped to `activeTab = 'calendar'`.
2. **Calendar Layout & Month Navigation**:
   - Full-screen, responsive monthly grid (Monday through Sunday columns).
   - Month navigation header: Previous Month, Next Month, "Today" jump button, and localized Month/Year display.
3. **Milestone Timeline Bars**:
   - Display milestones that overlap the viewed month using their derived target timeframe (`target_start_at` to `target_end_at`).
   - Render multi-day continuous horizontal timeline bars spanning across day cells/weeks.
   - Display milestone title, progress percentage / task counts, and click navigation to the milestone's filtered board view.
4. **Task Scheduling & Visual State Representation**:
   - **Completed Tasks (`status === 'done'`)**: Rendered on their `changed_at` date with completion styling.
   - **Active Tasks with Target (`status !== 'done'` with `target_at`)**: Rendered on their `target_at` date at full opacity (1.0).
   - **Active Tasks without Target (`status !== 'done'` without `target_at`)**: Rendered on their `changed_at` date at 0.75 opacity.
   - **State Dot Indicator**: Prefix each task chip/title with a solid colored dot matching its column status color.
   - Clicking a task chip opens the global `TaskDetailModal`.
5. **Filter Integration**:
   - Calendar items reactively filter based on active search queries, tags, and milestone selections in `store.ts`.

## Acceptance Criteria
- [ ] Add `#calendar` route to `router.ts` and Calendar tab link to `Header.tsx`
- [ ] Implement responsive full-screen `CalendarView` component with monthly grid layout
- [ ] Add month navigation controls (Previous, Next, Today) with current month/year indicator
- [ ] Render multi-day Milestone timeline bars spanning across day cells based on derived timeframe
- [ ] Render single-day task chips on appropriate dates (target_at for active, changed_at for done/untargeted)
- [ ] Apply 0.75 opacity for untargeted active tasks and full opacity for targeted tasks
- [ ] Display colored dot indicator matching column status color on task chips
- [ ] Connect task clicks to `TaskDetailModal` and milestone clicks to milestone view/filters
- [ ] Respect global search, tag, and milestone filter state
- [ ] Add Playwright E2E test suite covering calendar navigation, milestone spans, and task chips
