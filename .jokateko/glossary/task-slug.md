+++
title = 'Task slug'
tags = ['backend']
summary = "A task's ID and file name: `YYMMDD-<slug>`, where `YYMMDD` is the UTC creation date and the slug is the lowercase, hyphenated title cut to 40 characters, for example `261005-fix-login-redirect`."
+++

- Milestones use the same `YYMMDD-slug` pattern. Strategies and glossary terms use only `slug`.
- Every ID must match `^[a-z0-9][a-z0-9-]{0,99}$`, so it is always a safe single file name (no path traversal).
- If a generated ID already exists, a `-2`, `-3`, … suffix is added. A duplicate explicit ID is rejected.
- IDs never change when the title changes. References (`dependencies`, `milestone`) use the ID.

See the strategy *Workspace and entity file format*.
