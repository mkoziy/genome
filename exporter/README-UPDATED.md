# SNP Downloader - Updated Implementation Plan

## Overview
A Go application to download significant SNPs from public databases (ClinVar, dbSNP, OpenSNP) with calculated significance scores, stored in SQLite for later use in a 23andMe genome analysis SPA.

## Key Technologies

- **Language**: Go 1.21+
- **Database**: SQLite 3 with **Bun ORM** (https://bun.uptrace.dev/)
- **Logging**: **go-pkgz/lgr** with **wide event pattern**
- **CLI**: cobra
- **Rate Limiting**: Token bucket, fixed window, exponential backoff

## What's Updated

### ✅ Bun ORM Integration
- All database models defined with Bun tags and relations
- Migration system using `bun/migrate`
- Query builder for complex joins and aggregations
- Transaction support for batch operations
- See: **001-database-schema-bun.md**

### ✅ Wide Event Logging
- Single comprehensive event per operation
- Structured JSON logging optimized for querying
- Tail sampling: keep all errors/slow requests, sample successes
- High-cardinality context: IDs, scores, genes, flags
- go-pkgz/lgr logger implementation
- See: **010-wide-event-logging.md**

## Document Structure

### Core Documents
1. **000-project-overview.md** - Architecture, goals, technology stack
2. **001-database-schema-bun.md** - Complete Bun ORM schema with models, migrations, queries
3. **002-core-models.md** - Go structs with Bun tags, validation, helpers
4. **003-rate-limiting.md** - Token bucket, exponential backoff, per-source configs
5. **004-clinvar-source.md** - ClinVar E-utilities integration, XML parsing
6. **005-012-remaining-features.md** - dbSNP, scoring, OpenSNP, pipeline, CLI, reporting
7. **010-wide-event-logging.md** - Complete wide event pattern implementation

### Old Document (Reference)
- **001-database-schema.md** - Original raw SQL version (superseded by Bun version)

## Quick Start Implementation Order

1. **Database & Models** (001-database-schema-bun.md + 002-core-models.md)
   - Set up Bun ORM connection
   - Define all models with relations
   - Create migrations
   - Test queries

2. **Logging Infrastructure** (010-wide-event-logging.md)
   - Set up go-pkgz/lgr
   - Create wide event containers
   - Implement sampling logic
   - Add context propagation

3. **Rate Limiting** (003-rate-limiting.md)
   - Implement token bucket
   - Add exponential backoff
   - Configure per-source limits

4. **Data Sources** (004-clinvar-source.md + 005-012)
   - ClinVar fetcher with E-utilities
   - dbSNP enrichment
   - OpenSNP community data
   - Each emits wide events

5. **Scoring Engine** (005-012-remaining-features.md)
   - Clinical scoring (0-40 points)
   - Research scoring (0-30 points)
   - Population scoring (0-20 points)
   - Functional scoring (0-10 points)

6. **Pipeline & CLI** (005-012-remaining-features.md)
   - Orchestrate fetchers
   - Deduplicate SNPs
   - Calculate scores
   - Store with Bun transactions
   - CLI commands with cobra

## Bun ORM Examples

### Query SNP with relations
```go
snp := new(models.SNP)
err := db.NewSelect().
    Model(snp).
    Where("rsid = ?", "rs429358").
    Relation("Significance").
    Relation("ClinicalData").
    Relation("Phenotypes").
    Scan(ctx)
```

### Batch upsert
```go
_, err := db.NewInsert().
    Model(&snps).
    On("CONFLICT (rsid) DO UPDATE").
    Set("updated_at = CURRENT_TIMESTAMP").
    Exec(ctx)
```

### Complex join query
```go
var snps []*models.SNP
err := db.NewSelect().
    Model(&snps).
    Relation("Significance").
    Where("sig.total_score >= ?", 60).
    OrderExpr("sig.total_score DESC").
    Limit(100).
    Scan(ctx)
```

## Wide Event Logging Examples

### Fetcher operation
```go
event := logging.NewWideEvent(requestID, "fetch_clinvar_snp")
event.Source = &logging.SourceContext{Name: "clinvar"}
event.SNP = &logging.SNPContext{RsID: rsID}

startTime := time.Now()
defer func() {
    event.Complete(startTime, err)
    logging.EmitEvent(event)
}()

// ... do work, enrich event ...
event.Source.RequestCount++
event.SNP.SignificanceScore = 85.5
```

### Sampling configuration
```yaml
logging:
  sampling:
    error_rate: 1.0          # Keep all errors
    slow_threshold_ms: 5000
    slow_rate: 1.0           # Keep all slow requests
    success_rate: 0.05       # Keep 5% of successes
    vip_rate: 1.0            # Keep VIP operations (high scores, critical genes)
```

## Key Design Decisions

### Significance Scoring (0-100)
- **Clinical** (40%): Pathogenic=40, Likely Pathogenic=30, Risk Factor=25
- **Research** (30%): Based on PubMed citations, high-impact studies
- **Population** (20%): Based on minor allele frequency
- **Functional** (10%): Protein-changing variants score highest

### Translation Ready
- Separate `snp_translations` and `phenotype_translations` tables
- Fields: `language_code`, `field_name`, `translated_text`, `verified`
- Can be populated later without schema changes

### Rate Limiting Per Source
- **ClinVar/dbSNP**: 3 req/sec (no key), 10 req/sec (with key)
- **OpenSNP**: 1 req/sec (conservative)
- **SNPedia**: 1 req/5 sec (polite scraping)

## Performance Targets

- **Download**: 10,000 SNPs in 1-3 hours
- **rsID Lookup**: <1ms
- **Batch Insert**: ~100 SNPs/second
- **Database Size**: ~100MB for 10K SNPs
- **Memory**: <500MB peak

## Success Criteria

✅ 10,000+ significant SNPs downloaded
✅ 95%+ have clinical annotations  
✅ 90%+ have population data
✅ 80%+ have research references
✅ All scores calculated correctly
✅ Database <200MB
✅ Wide events emitted for all operations
✅ Sampling reduces log volume 95% while keeping critical events
✅ Bun queries perform well
✅ Ready for 23andMe integration

## Timeline Estimate

- **Week 1**: Database (Bun), Models, Logging (wide events)
- **Week 2**: Rate Limiting, ClinVar, dbSNP sources
- **Week 3**: Scoring Engine, OpenSNP, Pipeline
- **Week 4**: CLI, Error Handling, Stats, Docs
- **Week 5**: Testing, debugging, optimization
- **Week 6**: Documentation, examples, release

**Total**: 6 weeks for MVP

## Dependencies

```go
require (
    github.com/uptrace/bun v1.1.16
    github.com/uptrace/bun/dialect/sqlitedialect v1.1.16
    github.com/uptrace/bun/driver/sqliteshim v1.1.16
    github.com/uptrace/bun/extra/bundebug v1.1.16
    github.com/spf13/cobra v1.8.0
    github.com/spf13/viper v1.18.2
    github.com/go-pkgz/lgr v0.11.1
    github.com/google/uuid v1.5.0
    gopkg.in/yaml.v3 v3.0.1
)
```

## Next Steps

1. Review all feature documents
2. Set up Go project structure
3. Start with 001-database-schema-bun.md (Bun setup)
4. Implement 010-wide-event-logging.md (logging foundation)
5. Continue through features in order
6. Each document is AI-agent ready for independent implementation

## Questions?

Each feature document contains:
- Complete specifications
- Code examples
- Success criteria
- Testing guidelines
- Dependencies

Start with any document and implement independently!
