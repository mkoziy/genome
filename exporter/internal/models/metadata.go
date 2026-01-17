package models

import (
	"time"

	"github.com/uptrace/bun"
)

// SourceMetadata stores information about a data source.
type SourceMetadata struct {
	bun.BaseModel `bun:"table:data_sources,alias:ds"`

	ID           int64      `bun:"id,pk,autoincrement" json:"id"`
	SourceName   string     `bun:"source_name,unique,notnull" json:"source_name"`
	SourceURL    string     `bun:"source_url,notnull" json:"source_url"`
	APIVersion   *string    `bun:"api_version" json:"api_version,omitempty"`
	Description  *string    `bun:"description" json:"description,omitempty"`
	TermsOfUse   *string    `bun:"terms_of_use" json:"terms_of_use,omitempty"`
	LastAccessed *time.Time `bun:"last_accessed" json:"last_accessed,omitempty"`
	IsActive     bool       `bun:"is_active,default:true" json:"is_active"`
	CreatedAt    time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
}

// DownloadMetadata tracks download runs and their outcomes.
type DownloadMetadata struct {
	bun.BaseModel `bun:"table:download_metadata,alias:dm"`

	ID             int64      `bun:"id,pk,autoincrement" json:"id"`
	RunID          string     `bun:"run_id,unique,notnull" json:"run_id"`
	Source         string     `bun:"source,notnull" json:"source"`
	StartTime      time.Time  `bun:"start_time,notnull" json:"start_time"`
	EndTime        *time.Time `bun:"end_time" json:"end_time,omitempty"`
	Status         string     `bun:"status,notnull" json:"status"`
	SNPsDownloaded int        `bun:"snps_downloaded,default:0" json:"snps_downloaded"`
	SNPsUpdated    int        `bun:"snps_updated,default:0" json:"snps_updated"`
	SNPsSkipped    int        `bun:"snps_skipped,default:0" json:"snps_skipped"`
	ErrorsCount    int        `bun:"errors_count,default:0" json:"errors_count"`
	ErrorLog       *string    `bun:"error_log" json:"error_log,omitempty"`
	ConfigSnapshot *string    `bun:"config_snapshot" json:"config_snapshot,omitempty"`
	CreatedAt      time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
}
