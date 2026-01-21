package history

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/robottwo/bishop/pkg/reverse"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type HistoryManager struct {
	db *gorm.DB
}

type HistoryEntry struct {
	ID        uint      `gorm:"primarykey"`
	CreatedAt time.Time `gorm:"index;index:idx_dir_created,priority:2"`
	UpdatedAt time.Time `gorm:"index"`

	Command   string `gorm:"index"`
	Directory string `gorm:"index:idx_dir_created,priority:1"`
	SessionID string `gorm:"index"`
	ExitCode  sql.NullInt32
}

func NewHistoryManager(dbFilePath string) (*HistoryManager, error) {
	// NFS-optimized connection string with PRAGMA settings
	// - foreign_keys(1): Enable foreign key constraints (disabled by default)
	// - busy_timeout(5000): 5 second timeout for NFS network latency
	// - synchronous(1): NORMAL mode for durability/performance balance
	// - cache_size(-20000): 20MB cache to reduce NFS I/O operations
	// - temp_store(2): MEMORY - keeps temp files out of NFS
	connectionString := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=synchronous(1)&_pragma=cache_size(-20000)&_pragma=temp_store(2)", dbFilePath)

	db, err := gorm.Open(sqlite.Open(connectionString), &gorm.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening database")
		return nil, err
	}

	if err := db.AutoMigrate(&HistoryEntry{}); err != nil {
		return nil, err
	}

	// Configure connection pool for SQLite optimization
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// SQLite serializes writes anyway, so multiple connections add overhead
	sqlDB.SetMaxOpenConns(1)
	// Minimal pooling for file-based DB
	sqlDB.SetMaxIdleConns(1)
	// Reasonable connection lifetime
	sqlDB.SetConnMaxLifetime(time.Hour)

	return &HistoryManager{
		db: db,
	}, nil
}

// Close closes the database connection. This should be called when the
// HistoryManager is no longer needed, especially in tests to allow cleanup
// of temporary database files on Windows.
func (historyManager *HistoryManager) Close() error {
	sqlDB, err := historyManager.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// GetDB returns the underlying GORM database connection.
// This allows other packages (like coach) to use the same database.
func (historyManager *HistoryManager) GetDB() *gorm.DB {
	return historyManager.db
}

func (historyManager *HistoryManager) StartCommand(command string, directory string, sessionID string) (*HistoryEntry, error) {
	entry := HistoryEntry{
		Command:   command,
		Directory: directory,
		SessionID: sessionID,
	}

	result := historyManager.db.Create(&entry)
	if result.Error != nil {
		return nil, result.Error
	}

	return &entry, nil
}

func (historyManager *HistoryManager) FinishCommand(entry *HistoryEntry, exitCode int) (*HistoryEntry, error) {
	entry.ExitCode = sql.NullInt32{Int32: int32(exitCode), Valid: true}

	result := historyManager.db.Save(entry)
	if result.Error != nil {
		return nil, result.Error
	}

	return entry, nil
}

func (historyManager *HistoryManager) GetRecentEntries(directory string, limit int) ([]HistoryEntry, error) {
	var entries []HistoryEntry
	var db = historyManager.db
	if directory != "" {
		db = db.Where("directory = ?", directory)
	}
	result := db.Order("created_at desc").Limit(limit).Find(&entries)
	if result.Error != nil {
		return nil, result.Error
	}

	reverse.Reverse(entries)
	return entries, nil
}

// GetAllEntries returns all history entries ordered by creation time (newest first)
func (historyManager *HistoryManager) GetAllEntries() ([]HistoryEntry, error) {
	var entries []HistoryEntry
	result := historyManager.db.Order("created_at desc").Find(&entries)
	if result.Error != nil {
		return nil, result.Error
	}
	return entries, nil
}

func (historyManager *HistoryManager) DeleteEntry(id uint) error {
	result := historyManager.db.Delete(&HistoryEntry{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("no history entry found with id %d", id)
	}

	return nil
}

func (historyManager *HistoryManager) ResetHistory() error {
	result := historyManager.db.Exec("DELETE FROM history_entries")
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (historyManager *HistoryManager) GetRecentEntriesByPrefix(prefix string, limit int) ([]HistoryEntry, error) {
	var entries []HistoryEntry
	result := historyManager.db.Where("command LIKE ?", prefix+"%").
		Order("created_at desc").
		Limit(limit).
		Find(&entries)
	if result.Error != nil {
		return nil, result.Error
	}

	return entries, nil
}

// GetEntriesSince returns all history entries created after the given time, ordered by creation time (oldest first)
func (historyManager *HistoryManager) GetEntriesSince(since time.Time) ([]HistoryEntry, error) {
	var entries []HistoryEntry
	result := historyManager.db.Where("created_at >= ?", since).
		Order("created_at asc").
		Find(&entries)
	if result.Error != nil {
		return nil, result.Error
	}

	return entries, nil
}
