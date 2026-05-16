# Danmaku Control

`douyu-streamer` now supports basic Douyu danmaku control commands.

## Current commands

Assuming the configured prefix is `#`:

- `#next`
  switch to the next media item immediately

- `#reload`
  reload the media library and restart playback from the new queue

- `#<number>`
  switch immediately to the 1-based media index in the current queue

Examples:

```text
#next
#reload
#3
```

## Notes

- The queue order is based on the current scanned media list.
- `#3` means the third item in the current queue, not necessarily “episode 3”
  unless your directory contents are already ordered that way.
- This is the first minimal implementation. It favors direct control and
  predictable behavior over more complex scheduling rules.
