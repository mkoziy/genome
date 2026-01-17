# 001 - Database Schema with Bun ORM

## Feature Overview
Design and implement the SQLite database schema using Bun ORM that will store SNP data, clinical annotations, research references, and support future translation capabilities.

## Goals
- Create normalized schema for SNP data using Bun models
- Support efficient lookups by rsID (23andMe use case)
- Translation-ready design
- Track data provenance
- Enable incremental updates
- Leverage Bun's query builder and migrations

## Bun ORM Setup

### Installation
```go
// go.mod
require (
    github.com/uptrace/bun v1.1.16
    github.com/uptrace/bun/dialect/sqlitedialect v1.1.16
    github.com/uptrace/bun/driver/sqliteshim v1.1.16
    github.com/uptrace/bun/extra/bundebug v1.1.16
)
```

### Database Connection
```go
package database

import (
    "database/sql"
    
    "github.com/uptrace/bun"
    "github.com/uptrace/bun/dialect/sqlitedialect"
    "github.com/uptrace/bun/driver/sqliteshim"
    "github.com/uptrace/bun/extra/bundebug"
)

func NewDB(dsn string, debug bool) (*bun.DB, error) {
    sqldb, err := sql.Open(sqliteshim.ShimName, dsn)
    if err != nil {
        return nil, err
    }
    
    db := bun.NewDB(sqldb, sqlitedialect.New())
    
    // Add query hook for debugging
    if debug {
        db.AddQueryHook(bundebug.NewQueryHook(
            bundebug.WithVerbose(true),
        ))
    }
    
    // SQLite optimizations
    _, err = db.Exec(`
        PRAGMA journal_mode = WAL;
        PRAGMA synchronous = NORMAL;
        PRAGMA foreign_keys = ON;
        PRAGMA cache_size = -64000;
    `)
    
    return db, err
}
```

## Bun Models

### models/snp.go

```go
package models

import (
    "time"
    
    "github.com/uptrace/bun"
)

// SNP represents a Single Nucleotide Polymorphism
type SNP struct {
    bun.BaseModel `bun:"table:snps,alias:s"`
    
    ID                int64       `bun:"id,pk,autoincrement"`
    RsID              string      `bun:"rsid,unique,notnull"`
    Chromosome        string      `bun:"chromosome,notnull"`
    Position          int64       `bun:"position,notnull"`
    ReferenceAllele   string      `bun:"reference_allele,notnull"`
    AlternateAlleles  []string    `bun:"alternate_alleles,type:json,notnull"`
    GeneSymbol        *string     `bun:"gene_symbol"`
    GeneID            *string     `bun:"gene_id"`
    VariantType       VariantType `bun:"variant_type,notnull"`
    FunctionalClass   *FunctionalClass `bun:"functional_class"`
    CreatedAt         time.Time   `bun:"created_at,nullzero,notnull,default:current_timestamp"`
    UpdatedAt         time.Time   `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
    
    // Relations (not stored, loaded via joins)
    Significance      *Significance      `bun:"rel:has-one,join:id=snp_id"`
    ClinicalData      []*ClinicalData    `bun:"rel:has-many,join:id=snp_id"`
    Phenotypes        []*Phenotype       `bun:"rel:has-many,join:id=snp_id"`
    References        []*Reference       `bun:"rel:has-many,join:id=snp_id"`
    PopulationData    []*PopulationFreq  `bun:"rel:has-many,join:id=snp_id"`
}

// BeforeUpdate hook
func (s *SNP) BeforeUpdate(ctx context.Context, query *bun.UpdateQuery) error {
    s.UpdatedAt = time.Now()
    return nil
}
```

### models/significance.go

```go
package models

import (
    "time"
    
    "github.com/uptrace/bun"
)

// Significance represents the calculated significance score
type Significance struct {
    bun.BaseModel `bun:"table:snp_significance,alias:sig"`
    
    ID              int64          `bun:"id,pk,autoincrement"`
    SNPID           int64          `bun:"snp_id,notnull,unique"`
    TotalScore      float64        `bun:"total_score,notnull"`
    ClinicalScore   float64        `bun:"clinical_score,notnull"`
    ResearchScore   float64        `bun:"research_score,notnull"`
    PopulationScore float64        `bun:"population_score,notnull"`
    FunctionalScore float64        `bun:"functional_score,notnull"`
    ScoreDetails    ScoreBreakdown `bun:"score_details,type:json"`
    CalculatedAt    time.Time      `bun:"calculated_at,nullzero,notnull,default:current_timestamp"`
    
    // Relation
    SNP             *SNP           `bun:"rel:belongs-to,join:snp_id=id"`
}

// ScoreBreakdown stored as JSON
type ScoreBreakdown struct {
    ClinicalDetails    ClinicalScoring    `json:"clinical"`
    ResearchDetails    ResearchScoring    `json:"research"`
    PopulationDetails  PopulationScoring  `json:"population"`
    FunctionalDetails  FunctionalScoring  `json:"functional"`
}

type ClinicalScoring struct {
    HasPathogenic      bool    `json:"has_pathogenic"`
    ReviewStatusScore  float64 `json:"review_status_score"`
    ConditionCount     int     `json:"condition_count"`
}

type ResearchScoring struct {
    PubmedCount        int     `json:"pubmed_count"`
    CitationTotal      int     `json:"citation_total"`
    HighImpactStudies  int     `json:"high_impact_studies"`
}

type PopulationScoring struct {
    MaxMAF             float64 `json:"max_maf"`
    PopulationCount    int     `json:"population_count"`
}

type FunctionalScoring struct {
    IsProteinChanging  bool    `json:"is_protein_changing"`
    IsRegulatory       bool    `json:"is_regulatory"`
}
```

### models/clinical.go

```go
package models

import (
    "time"
    
    "github.com/uptrace/bun"
)

// ClinicalData represents clinical significance information
type ClinicalData struct {
    bun.BaseModel `bun:"table:snp_clinical,alias:c"`
    
    ID                   int64                `bun:"id,pk,autoincrement"`
    SNPID                int64                `bun:"snp_id,notnull"`
    ClinicalSignificance ClinicalSignificance `bun:"clinical_significance,notnull"`
    ReviewStatus         ReviewStatus         `bun:"review_status,notnull"`
    ConditionName        string               `bun:"condition_name,notnull"`
    ConditionID          *string              `bun:"condition_id"`
    InheritancePattern   *string              `bun:"inheritance_pattern"`
    Penetrance           *string              `bun:"penetrance"`
    AlleleOrigin         *string              `bun:"allele_origin"`
    Source               DataSource           `bun:"source,notnull"`
    SourceID             *string              `bun:"source_id"`
    LastEvaluated        *time.Time           `bun:"last_evaluated"`
    CreatedAt            time.Time            `bun:"created_at,nullzero,notnull,default:current_timestamp"`
    
    // Relation
    SNP                  *SNP                 `bun:"rel:belongs-to,join:snp_id=id"`
}
```

### models/phenotype.go

```go
package models

import (
    "time"
    
    "github.com/uptrace/bun"
)

// Phenotype represents a trait or condition associated with SNP
type Phenotype struct {
    bun.BaseModel `bun:"table:snp_phenotypes,alias:p"`
    
    ID                int64      `bun:"id,pk,autoincrement"`
    SNPID             int64      `bun:"snp_id,notnull"`
    PhenotypeName     string     `bun:"phenotype_name,notnull"`
    PhenotypeID       *string    `bun:"phenotype_id"`
    AssociationType   string     `bun:"association_type,notnull"`
    OddsRatio         *float64   `bun:"odds_ratio"`
    ConfidenceInterval *string   `bun:"confidence_interval"`
    PValue            *float64   `bun:"p_value"`
    StudyType         *string    `bun:"study_type"`
    Source            DataSource `bun:"source,notnull"`
    CreatedAt         time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp"`
    
    // Relation
    SNP               *SNP       `bun:"rel:belongs-to,join:snp_id=id"`
}
```

### models/reference.go

```go
package models

import (
    "time"
    
    "github.com/uptrace/bun"
)

// Reference represents a research paper or study
type Reference struct {
    bun.BaseModel `bun:"table:snp_references,alias:r"`
    
    ID              int64     `bun:"id,pk,autoincrement"`
    SNPID           int64     `bun:"snp_id,notnull"`
    PubmedID        *string   `bun:"pubmed_id"`
    Title           *string   `bun:"title"`
    Authors         *string   `bun:"authors"`
    Journal         *string   `bun:"journal"`
    PublicationYear *int      `bun:"publication_year"`
    DOI             *string   `bun:"doi"`
    URL             *string   `bun:"url"`
    CitationCount   int       `bun:"citation_count,default:0"`
    Abstract        *string   `bun:"abstract"`
    CreatedAt       time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
    
    // Relation
    SNP             *SNP      `bun:"rel:belongs-to,join:snp_id=id"`
}
```

### models/population.go

```go
package models

import (
    "time"
    
    "github.com/uptrace/bun"
)

// PopulationFreq represents allele frequency in a population
type PopulationFreq struct {
    bun.BaseModel `bun:"table:snp_populations,alias:pop"`
    
    ID              int64      `bun:"id,pk,autoincrement"`
    SNPID           int64      `bun:"snp_id,notnull"`
    PopulationCode  string     `bun:"population_code,notnull"`
    PopulationName  *string    `bun:"population_name"`
    Allele          string     `bun:"allele,notnull"`
    Frequency       float64    `bun:"frequency,notnull"`
    AlleleCount     *int       `bun:"allele_count"`
    AlleleNumber    *int       `bun:"allele_number"`
    HomozygoteCount *int       `bun:"homozygote_count"`
    Source          DataSource `bun:"source,notnull"`
    CreatedAt       time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp"`
    
    // Relation
    SNP             *SNP       `bun:"rel:belongs-to,join:snp_id=id"`
}
```

### models/translation.go

```go
package models

import (
    "time"
    
    "github.com/uptrace/bun"
)

// Translation represents translated content
type Translation struct {
    bun.BaseModel `bun:"table:snp_translations,alias:t"`
    
    ID             int64     `bun:"id,pk,autoincrement"`
    SNPID          int64     `bun:"snp_id,notnull"`
    LanguageCode   string    `bun:"language_code,notnull"`
    FieldName      string    `bun:"field_name,notnull"`
    TranslatedText string    `bun:"translated_text,notnull"`
    Translator     *string   `bun:"translator"`
    TranslatedAt   time.Time `bun:"translated_at,nullzero,notnull,default:current_timestamp"`
    Verified       bool      `bun:"verified,default:false"`
    
    // Relation
    SNP            *SNP      `bun:"rel:belongs-to,join:snp_id=id"`
}

// PhenotypeTranslation represents translated phenotype names
type PhenotypeTranslation struct {
    bun.BaseModel `bun:"table:phenotype_translations,alias:pt"`
    
    ID             int64     `bun:"id,pk,autoincrement"`
    PhenotypeID    int64     `bun:"phenotype_id,notnull"`
    LanguageCode   string    `bun:"language_code,notnull"`
    TranslatedName string    `bun:"translated_name,notnull"`
    Translator     *string   `bun:"translator"`
    TranslatedAt   time.Time `bun:"translated_at,nullzero,notnull,default:current_timestamp"`
    Verified       bool      `bun:"verified,default:false"`
    
    // Relation
    Phenotype      *Phenotype `bun:"rel:belongs-to,join:phenotype_id=id"`
}
```

### models/metadata.go

```go
package models

import (
    "time"
    
    "github.com/uptrace/bun"
)

// SourceMetadata represents information about a data source
type SourceMetadata struct {
    bun.BaseModel `bun:"table:data_sources,alias:ds"`
    
    ID           int64      `bun:"id,pk,autoincrement"`
    SourceName   string     `bun:"source_name,unique,notnull"`
    SourceURL    string     `bun:"source_url,notnull"`
    APIVersion   *string    `bun:"api_version"`
    Description  *string    `bun:"description"`
    TermsOfUse   *string    `bun:"terms_of_use"`
    LastAccessed *time.Time `bun:"last_accessed"`
    IsActive     bool       `bun:"is_active,default:true"`
    CreatedAt    time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp"`
}

// DownloadMetadata tracks download runs
type DownloadMetadata struct {
    bun.BaseModel `bun:"table:download_metadata,alias:dm"`
    
    ID             int64      `bun:"id,pk,autoincrement"`
    RunID          string     `bun:"run_id,unique,notnull"`
    Source         string     `bun:"source,notnull"`
    StartTime      time.Time  `bun:"start_time,notnull"`
    EndTime        *time.Time `bun:"end_time"`
    Status         string     `bun:"status,notnull"`
    SNPsDownloaded int        `bun:"snps_downloaded,default:0"`
    SNPsUpdated    int        `bun:"snps_updated,default:0"`
    SNPsSkipped    int        `bun:"snps_skipped,default:0"`
    ErrorsCount    int        `bun:"errors_count,default:0"`
    ErrorLog       *string    `bun:"error_log"`
    ConfigSnapshot *string    `bun:"config_snapshot"`
    CreatedAt      time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp"`
}
```

## Migrations with Bun

### migrations/main.go

```go
package migrations

import (
    "context"
    "fmt"
    
    "github.com/uptrace/bun"
    "github.com/uptrace/bun/migrate"
    
    "snp-downloader/internal/models"
)

var Migrations = migrate.NewMigrations()

func init() {
    // Migration 1: Create all tables
    Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
        models := []interface{}{
            (*models.SNP)(nil),
            (*models.Significance)(nil),
            (*models.ClinicalData)(nil),
            (*models.Phenotype)(nil),
            (*models.Reference)(nil),
            (*models.PopulationFreq)(nil),
            (*models.Translation)(nil),
            (*models.PhenotypeTranslation)(nil),
            (*models.SourceMetadata)(nil),
            (*models.DownloadMetadata)(nil),
        }
        
        for _, model := range models {
            _, err := db.NewCreateTable().
                Model(model).
                IfNotExists().
                Exec(ctx)
            if err != nil {
                return err
            }
        }
        
        return nil
    }, func(ctx context.Context, db *bun.DB) error {
        // Rollback: drop tables
        models := []interface{}{
            (*models.DownloadMetadata)(nil),
            (*models.SourceMetadata)(nil),
            (*models.PhenotypeTranslation)(nil),
            (*models.Translation)(nil),
            (*models.PopulationFreq)(nil),
            (*models.Reference)(nil),
            (*models.Phenotype)(nil),
            (*models.ClinicalData)(nil),
            (*models.Significance)(nil),
            (*models.SNP)(nil),
        }
        
        for _, model := range models {
            _, err := db.NewDropTable().
                Model(model).
                IfExists().
                Exec(ctx)
            if err != nil {
                return err
            }
        }
        
        return nil
    })
    
    // Migration 2: Create indexes
    Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
        indexes := []string{
            "CREATE INDEX IF NOT EXISTS idx_snps_chromosome_position ON snps(chromosome, position)",
            "CREATE INDEX IF NOT EXISTS idx_snps_gene_symbol ON snps(gene_symbol)",
            "CREATE INDEX IF NOT EXISTS idx_significance_score ON snp_significance(total_score DESC)",
            "CREATE INDEX IF NOT EXISTS idx_clinical_significance ON snp_clinical(clinical_significance)",
            "CREATE INDEX IF NOT EXISTS idx_clinical_condition ON snp_clinical(condition_name)",
            "CREATE INDEX IF NOT EXISTS idx_phenotypes_name ON snp_phenotypes(phenotype_name)",
            "CREATE INDEX IF NOT EXISTS idx_references_pubmed ON snp_references(pubmed_id)",
            "CREATE INDEX IF NOT EXISTS idx_populations_code ON snp_populations(population_code)",
            "CREATE INDEX IF NOT EXISTS idx_translations_snp_lang ON snp_translations(snp_id, language_code)",
        }
        
        for _, idx := range indexes {
            if _, err := db.ExecContext(ctx, idx); err != nil {
                return err
            }
        }
        
        return nil
    }, func(ctx context.Context, db *bun.DB) error {
        // Rollback: drop indexes
        indexes := []string{
            "DROP INDEX IF EXISTS idx_snps_chromosome_position",
            "DROP INDEX IF EXISTS idx_snps_gene_symbol",
            "DROP INDEX IF EXISTS idx_significance_score",
            "DROP INDEX IF EXISTS idx_clinical_significance",
            "DROP INDEX IF EXISTS idx_clinical_condition",
            "DROP INDEX IF EXISTS idx_phenotypes_name",
            "DROP INDEX IF EXISTS idx_references_pubmed",
            "DROP INDEX IF EXISTS idx_populations_code",
            "DROP INDEX IF EXISTS idx_translations_snp_lang",
        }
        
        for _, idx := range indexes {
            if _, err := db.ExecContext(ctx, idx); err != nil {
                return err
            }
        }
        
        return nil
    })
}

// RunMigrations runs all pending migrations
func RunMigrations(ctx context.Context, db *bun.DB) error {
    migrator := migrate.NewMigrator(db, Migrations)
    
    if err := migrator.Init(ctx); err != nil {
        return err
    }
    
    group, err := migrator.Migrate(ctx)
    if err != nil {
        return err
    }
    
    if group.IsZero() {
        fmt.Println("No new migrations to run")
        return nil
    }
    
    fmt.Printf("Migrated to %s\n", group)
    return nil
}
```

## Example Queries with Bun

### Find SNP by rsID with all relations

```go
func GetSNPByRsID(ctx context.Context, db *bun.DB, rsID string) (*models.SNP, error) {
    snp := new(models.SNP)
    err := db.NewSelect().
        Model(snp).
        Where("rsid = ?", rsID).
        Relation("Significance").
        Relation("ClinicalData").
        Relation("Phenotypes").
        Relation("References").
        Relation("PopulationData").
        Scan(ctx)
    
    return snp, err
}
```

### Get top significant SNPs

```go
func GetTopSignificantSNPs(ctx context.Context, db *bun.DB, limit int) ([]*models.SNP, error) {
    var snps []*models.SNP
    err := db.NewSelect().
        Model(&snps).
        Relation("Significance").
        Relation("ClinicalData", func(q *bun.SelectQuery) *bun.SelectQuery {
            return q.Where("clinical_significance IN (?)", bun.In([]string{"pathogenic", "likely_pathogenic"}))
        }).
        Where("sig.total_score >= ?", 60).
        OrderExpr("sig.total_score DESC").
        Limit(limit).
        Scan(ctx)
    
    return snps, err
}
```

### Insert SNP with related data

```go
func InsertSNPWithData(ctx context.Context, db *bun.DB, snp *models.SNP, clinical []*models.ClinicalData, refs []*models.Reference) error {
    return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
        // Insert SNP
        _, err := tx.NewInsert().Model(snp).Exec(ctx)
        if err != nil {
            return err
        }
        
        // Set SNP ID on related data
        for _, c := range clinical {
            c.SNPID = snp.ID
        }
        for _, r := range refs {
            r.SNPID = snp.ID
        }
        
        // Insert clinical data
        if len(clinical) > 0 {
            _, err = tx.NewInsert().Model(&clinical).Exec(ctx)
            if err != nil {
                return err
            }
        }
        
        // Insert references
        if len(refs) > 0 {
            _, err = tx.NewInsert().Model(&refs).Exec(ctx)
            if err != nil {
                return err
            }
        }
        
        return nil
    })
}
```

### Batch upsert SNPs

```go
func UpsertSNPs(ctx context.Context, db *bun.DB, snps []*models.SNP) error {
    _, err := db.NewInsert().
        Model(&snps).
        On("CONFLICT (rsid) DO UPDATE").
        Set("chromosome = EXCLUDED.chromosome").
        Set("position = EXCLUDED.position").
        Set("reference_allele = EXCLUDED.reference_allele").
        Set("alternate_alleles = EXCLUDED.alternate_alleles").
        Set("gene_symbol = EXCLUDED.gene_symbol").
        Set("updated_at = CURRENT_TIMESTAMP").
        Exec(ctx)
    
    return err
}
```

## Success Criteria
- All tables created via Bun migrations
- Foreign keys enforced
- Indexes improve query performance >10x
- Migrations reversible
- No data loss on schema updates
- Ready for 23andMe rsID lookups (<10ms per query)
- Bun relations work correctly
- Batch operations efficient

## Next Feature
After completing this, proceed to **002 - Core Models and Data Structures** (already defined above with Bun tags).
