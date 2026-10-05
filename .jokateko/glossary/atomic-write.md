+++
title = 'Atomic write'
tags = ['backend']
summary = 'Jokateko never writes a file in place: it writes a temp file in the same directory, fsyncs it, and renames it over the target, so readers and the watcher only ever see the old or the new complete file.'
+++

Implemented in `internal/writer`: `.<name>.*.tmp` → write → `fsync` → `chmod` → `rename`. The temp file is removed on any error.

Every atomic write is also recorded in a short-lived **suppression cache** (path + content, 3 s in `serve`). When `fsnotify` reports the rename, the watcher compares the content and drops the event, so Jokateko does not re-ingest its own writes. An external edit with different content made in the same window is still ingested.
