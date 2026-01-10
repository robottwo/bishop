## 2025-02-23 - SQLite WAL Configuration
**Learning:** The pure Go SQLite driver (`glebarez/sqlite`) supports standard query parameters (like `?_pragma=journal_mode(WAL)`) in the DSN. This is the most reliable way to configure persistent performance settings like WAL mode and synchronous flags, ensuring they are applied at connection time.
**Action:** Always verify SQLite configuration via DSN parameters rather than assuming defaults or running raw PRAGMA commands, especially for `journal_mode`.
