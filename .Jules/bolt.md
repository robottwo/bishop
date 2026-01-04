# Bolt's Journal

## 2025-05-18 - Missing GORM Indexes
**Learning:** `gorm:"index"` must be explicitly added to fields used in `WHERE` clauses for optimal performance.
**Action:** Always check `WHERE` clauses against model definitions to ensure indexes exist.
