+++
title = 'Full-Screen Interactive Calendar View for Milestones and Tasks'
status = 'done'
priority = 'medium'
tags = ['feature', 'frontend']
summary = 'Full-screen interactive calendar view supporting monthly grid, weekly view with mobile snap-x column navigation, milestone timelines, scheduled task chips, and state filtering.'
dependencies = ['260904-task-and-milestone-timestamps-createdat-']
created_at = '2026-09-05T17:02:34Z'
changed_at = '2026-09-06T13:30:38Z'
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
- [x] Add `#calendar` route to `router.ts` and Calendar tab link to `Header.tsx`
- [x] Implement responsive full-screen `CalendarView` component with monthly grid layout
- [x] Add month navigation controls (Previous, Next, Today) with current month/year indicator
- [x] Render multi-day Milestone timeline bars spanning across day cells based on derived timeframe
- [x] Render single-day task chips on appropriate dates (target_at for active, changed_at for done/untargeted)
- [x] Apply 0.75 opacity for untargeted active tasks and full opacity for targeted tasks
- [x] Display colored dot indicator matching column status color on task chips
- [x] Connect task clicks to `TaskDetailModal` and milestone clicks to milestone view/filters
- [x] Respect global search, tag, and milestone filter state
- [x] Add Playwright E2E test suite covering calendar navigation, milestone spans, and task chips

## Notes

### [2026-09-06 06:40 UTC]

Implemented full-screen interactive Calendar View:
- Deep linking & anchor URLs (`#calendar`, `#calendar/YYYY-MM`, `#calendar/YYYY-Www`, `#calendar/YYYY-MM/task/<id>`, `#calendar/YYYY-MM/milestone/<id>`).
- Responsive month and week grid layout with week number indicators and Prev/Next/Today controls.
- Continuous multi-day milestone timeline bars segmented across week rows with progress % and Kanban milestone filter navigation.
- Single-day task chips positioned by `target_at` (full 1.0 opacity) or `changed_at` (0.75 opacity for untargeted), line-through on completed tasks, colored column status dots, and `+N more` popover for overflow.
- Day cell and hover `+` click for quick task creation with pre-filled `target_at`.
- Full integration with global FilterBar (search, tag, milestone, priority).
- Playwright E2E test suite passing across all scenarios.

### [2026-09-06 13:00 UTC]

Implemented requested user refinements:
- Navigated "+N more" directly to weekly view instead of popover to facilitate reading.
- Updated weekly view to show all scheduled tasks with larger typography and scrollable day cell layout.
- Fixed week view title number formatting (accurate ISO week calculation, Thursday month reference, removed "W" prefix).
- Added state filter column buttons to FilterBar alongside tags and priority filters.
- Added comprehensive i18n support for weekday names, month names, state filters, and week numbers.
- Verified and validated with 40/40 E2E Playwright tests and 60/60 unit tests.

### [2026-09-06 13:27 UTC]

Implemented mobile weekly column view enhancements per user request:
- Mobile (< md:) Weekly view renders as full-screen width day columns (`w-[calc(100vw-2.5rem)]` on mobile, `sm:w-96` on tablet).
- Enabled smooth horizontal scroll snapping with `snap-x snap-mandatory scroll-smooth` matching the Kanban board behavior.
- Added responsive quick-navigation day pill bar (`week-day-quick-nav`) on mobile showing day names, dates, task count badges, and current active day indicator.
- Included in-column milestone banners and empty-state placeholders on mobile week columns.
- Resolved Preact dayOfWeekIndex lookup bug and verified with Playwright mobile viewport E2E test.
- All 41 Playwright E2E tests, 60 unit tests, Biome checks, and Go tests pass cleanly. Built binary placed into `/home/bun/.local/bin/jokateko`.

## Completion Summary
- **Completed At:** 2026-09-06T13:30:38Z

### What Was Done
Implemented dedicated full-screen CalendarView component with responsive monthly and weekly views, milestone timeline spans, task chips scheduled by target_at/changed_at, mobile horizontal snap-x scroll columns with quick-nav day pill bar, state filter buttons in FilterBar, accurate ISO week title formatting, and comprehensive i18n support.

### Why / Rationale
Gives developers and teams a chronological perspective to track deadlines, multi-day milestones, and delivery cadence across mobile and desktop devices without breaking Spec-First, Tasks-as-Code principles.
