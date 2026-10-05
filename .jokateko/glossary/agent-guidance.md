+++
title = 'Agent guidance'
tags = ['backend', 'docs']
summary = 'The rules Jokateko gives AI agents automatically: the MCP server `instructions` (from `[mcp] instructions` plus per-column ownership and instructions) telling agents to work only through MCP tools and to follow the task workflow.'
+++

Every MCP client receives these instructions in the handshake, so no per-agent prompt setup is needed. The default text says:

1. Read the full task (`get_task`) before starting.
2. Tick criteria with `update_task_item` as you go.
3. Finish with `complete_task` (what was done and why).
4. Read tier-1 strategies before architectural decisions.
5. Change `.jokateko/` **only through MCP tools**, never with file-writing tools. Direct writes skip validation, cycle checks, timestamps and live UI updates.

Projects can replace the text with `[mcp] instructions` and add per-column rules with `handled_by` / `instructions` (see *Board state policy*). Repository-level agent files such as `AGENTS.md` add project-specific rules on top.
