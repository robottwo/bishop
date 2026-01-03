## 2024-01-01 - Dynamic CLI Help Generation
**Learning:** Hardcoding flag lists in custom help text creates a maintenance burden and a source of truth mismatch.
**Action:** Always use `flag.VisitAll` to dynamically generate help text, even when styling it manually. Grouping by usage string is a reliable way to handle aliases without explicit metadata.

## 2024-05-22 - Contextual Empty States
**Learning:** Generic "no results" messages in search interfaces are a missed opportunity for guidance. Users often don't realize why a search failed (typo vs. filter).
**Action:** Always echo the search query in the empty state (e.g., "No matches for 'xyz'") and provide context-aware tips based on active filters (e.g., "Try clearing the directory filter").
