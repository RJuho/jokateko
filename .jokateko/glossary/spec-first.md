+++
title = 'Spec-First'
tags = ['docs']
summary = 'Write down what "done" means (acceptance criteria) and which rules apply (strategies) before any code is written, so humans and agents work against the same explicit spec.'
+++

In Jokateko, Spec-First is enforced through the workflow, not only recommended:

- A task's body holds its spec and `- [ ]` acceptance criteria. The body is editable only in `[board] editable_states` (default: Backlog). Once work starts, the spec is frozen, and new findings go into notes.
- `complete_task` refuses to finish a task while any criterion is unchecked, and it requires *what was done* and *why*.
- Architectural rules live in tiered strategies that agents read before deciding (see *Strategy tier*).

Related: *Tasks-as-Code*, *Board state policy*.
