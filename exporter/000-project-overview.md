# SNP Downloader Project Overview

## Project Goal
Build a Go application that downloads significant Single Nucleotide Polymorphisms (SNPs) from free public APIs/databases, stores them in SQLite with proper schema for later translation support, and prepares the data for use in a local-first SPA that parses 23andMe genome files.

## Architecture Principles

### Modular Design
- Each data source is a separate module/package
- Database layer abstracted for easy schema changes
- Translation-ready schema from day one
- Clean separation: fetcher → processor → storage

### Data Sources (Free APIs)
1. **ClinVar** (NCBI) - Clinical significance data
2. **dbSNP** (NCBI) - Reference SNP data
3. **OpenSNP** - Community-contributed data
4. **PharmGKB** (if API available) - Pharmacogenomic data
5. **SNPedia** (web scraping fallback) - Curated information

### Significance Scoring System
Calculate a weighted score (0-100) based on:
- **Clinical Significance** (40 points)
  - Pathogenic: 40
  - Likely Pathogenic: 30
  - Risk Factor: 25
  - Uncertain: 10
  - Benign/Likely Benign: 5
  
- **Research Evidence** (30 points)
  - Number of citations/studies (scaled)
  - PubMed references count
  
- **Population Impact** (20 points)
  - Minor Allele Frequency (MAF)
  - Higher frequency = higher impact
  
- **Functional Impact** (10 points)
  - Protein-changing variants
  - Regulatory region variants
  - Gene expression effects

### Database Schema Strategy

#### Core Tables
- `snps` - Main SNP data (rsID, position, alleles)
- `snp_significance` - Calculated significance scores
- `snp_clinical` - Clinical annotations
- `snp_phenotypes` - Associated traits/conditions
- `snp_references` - Research paper links
- `snp_translations` - Multi-language support (added later)
- `snp_populations` - Population frequency data
- `data_sources` - Track where data came from
- `download_metadata` - Download history and stats

#### Translation-Ready Design
- Separate `snp_translations` table with:
  - `snp_id`, `language_code`, `field_name`, `translated_text`
  - Supports translating: name, description, summary, phenotype_name

### Rate Limiting Strategy
- Configurable delays per API source
- Token bucket algorithm for API calls
- Respect HTTP 429 responses
- Exponential backoff on errors

### Project Structure
```
snp-downloader/
├── cmd/
│   └── downloader/          # Main application
├── internal/
│   ├── database/            # SQLite operations
│   ├── sources/             # Data source implementations
│   │   ├── clinvar/
│   │   ├── dbsnp/
│   │   ├── opensnp/
│   │   └── pharmgkb/
│   ├── models/              # Data structures
│   ├── scoring/             # Significance calculation
│   ├── ratelimit/           # Rate limiting utilities
│   └── processor/           # Data processing pipeline
├── migrations/              # SQL schema migrations
├── config/                  # Configuration files
└── docs/                    # Feature documentation
```

## Implementation Order

1. **001** - Database schema and migrations
2. **002** - Core models and data structures
3. **003** - Rate limiting system
4. **004** - ClinVar data source (start with most important)
5. **005** - dbSNP data source
6. **006** - Significance scoring engine
7. **007** - OpenSNP data source
8. **008** - Data processing pipeline
9. **009** - CLI interface and configuration
10. **010** - Error handling and logging
11. **011** - Statistics and reporting
12. **012** - Documentation and usage guide

## Success Criteria
- Downloads 10,000+ significant SNPs
- Each SNP has significance score
- Clinical annotations present where available
- Research links stored
- Database optimized for 23andMe rsID lookups
- Ready for translation without schema changes
- Respects all API rate limits
- Recoverable from interruptions

## Non-Goals (Out of Scope)
- Web UI (that's the separate SPA)
- Real-time updates
- User authentication
- 23andMe file parsing (separate component)
- Automatic translation (manual process later)

## Technical Stack
- **Language**: Go 1.21+
- **Database**: SQLite 3
- **ORM**: Bun (https://bun.uptrace.dev/)
- **HTTP Client**: Standard library + retry logic
- **CLI Framework**: cobra or similar
- **Configuration**: YAML or TOML
- **Logging**: go-pkgz/lgr with wide event pattern

## Next Steps
Review each feature document (001-012) for detailed implementation specifications. Each document is self-contained and can be implemented by an AI agent independently.
