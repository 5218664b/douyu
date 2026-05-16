# Preprocess

This directory is reserved for offline media preprocessing helpers.

The Pi-first rebuild assumes source media should be normalized ahead of runtime
so the streaming path can stay close to `ffmpeg -c copy` where possible.
