package analytics

import (
	"fmt"
	"os"
	"time"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"mvdan.cc/sh/v3/interp"
)

type AnalyticsManager struct {
	db     *gorm.DB
	Runner *interp.Runner
	Logger *zap.Logger
}

type AnalyticsEntry struct {
	ID        uint      `gorm:"primarykey"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`

	Input      string
	Prediction string
	Actual     string
}

func NewAnalyticsManager(dbFilePath string) (*AnalyticsManager, error) {
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

	if err := db.AutoMigrate(&AnalyticsEntry{}); err != nil {
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

	// Enable WAL mode for better NFS performance and concurrent readers
	if err := db.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	return &AnalyticsManager{
		db: db,
	}, nil
}

// Close closes the database connection. This should be called when the
// AnalyticsManager is no longer needed, especially in tests to allow cleanup
// of temporary database files on Windows.
func (analyticsManager *AnalyticsManager) Close() error {
	if analyticsManager.db == nil {
		return nil
	}
	sqlDB, err := analyticsManager.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (analyticsManager *AnalyticsManager) NewEntry(input string, prediction string, actual string) error {
	entry := AnalyticsEntry{
		Input:      input,
		Prediction: prediction,
		Actual:     actual,
	}

	result := analyticsManager.db.Create(&entry)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (analyticsManager *AnalyticsManager) GetRecentEntries(limit int) ([]AnalyticsEntry, error) {
	var entries []AnalyticsEntry
	result := analyticsManager.db.Where("input <> '' AND actual NOT LIKE '#%'").Order("created_at desc").Limit(limit).Find(&entries)
	if result.Error != nil {
		return nil, result.Error
	}
	return entries, nil
}

func (analyticsManager *AnalyticsManager) ResetAnalytics() error {
	result := analyticsManager.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&AnalyticsEntry{})
	return result.Error
}

func (analyticsManager *AnalyticsManager) DeleteEntry(id uint) error {
	result := analyticsManager.db.Delete(&AnalyticsEntry{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("entry not found")
	}
	return nil
}

func (analyticsManager *AnalyticsManager) GetTotalCount() (int64, error) {
	var count int64
	result := analyticsManager.db.Model(&AnalyticsEntry{}).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}
