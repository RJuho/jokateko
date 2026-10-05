+++
title = 'Strategy tier'
tags = ['docs']
summary = 'The 1–3 level of an architectural strategy (1 core invariants, 2 domain patterns and data flow, 3 implementation conventions), used for progressive disclosure: agents read summaries first and load full bodies only when relevant.'
+++

Tiers are configured in `[[strategies.tiers]]` (default titles: *Core Architecture & Tech Stack*, *Domain Logic & Data Flow*, *Code Conventions & UI Standards*) and set per strategy with `tier = N`.

**Progressive disclosure** keeps agent context small:
1. `list_strategies` returns only titles, tiers and summaries.
2. The agent fetches the full body (`get_strategy`) only for strategies that matter for the task.
3. Tier 1 is short and always relevant. It is also offered as the `jokateko://strategies/tier1` resource and in the `next_task` prompt.

Write tier-1 strategies as a handful of non-negotiables. Put detail in tier 2 and tier 3.
