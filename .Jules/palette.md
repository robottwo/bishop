## 2024-01-01 - Dynamic CLI Help Generation
**Learning:** Hardcoding flag lists in custom help text creates a maintenance burden and a source of truth mismatch.
**Action:** Always use `flag.VisitAll` to dynamically generate help text, even when styling it manually. Grouping by usage string is a reliable way to handle aliases without explicit metadata.
