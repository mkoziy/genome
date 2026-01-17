# 002 - Core Models and Data Structures

## Feature Overview
Define Go structs and data models that represent SNP data throughout the application. These models serve as the contract between data sources, processors, and the database layer.

## Goals
- Type-safe data structures
- Easy JSON marshaling/unmarshaling for APIs
- Database mapping support
- Validation logic
- Extensible for new fields

## Package Structure
```
internal/models/
├── snp.go              # Core SNP model
├── clinical.go         # Clinical annotations
├── phenotype.go        # Phenotype associations
├── reference.go        # Research references
├── population.go       # Population frequencies
├── significance.go     # Significance scoring
├── translation.go      # Translation support
├── source.go          # Data source metadata
├── validation.go       # Validation functions
└── types.go           # Common types and enums
```

## Core Models

### types.go - Common Types and Enums

```go
package models

import (
    "database/sql/driver"
    "encoding/json"
    "errors"
    "time"
)

// ClinicalSignificance represents the clinical impact
type ClinicalSignificance string

const (
    Pathogenic           ClinicalSignificance = "pathogenic"
    LikelyPathogenic     ClinicalSignificance = "likely_pathogenic"
    UncertainSignificance ClinicalSignificance = "uncertain_significance"
    LikelyBenign         ClinicalSignificance = "likely_benign"
    Benign               ClinicalSignificance = "benign"
    RiskFactor           ClinicalSignificance = "risk_factor"
    Protective           ClinicalSignificance = "protective"
    DrugResponse         ClinicalSignificance = "drug_response"
    Association          ClinicalSignificance = "association"
    Other                ClinicalSignificance = "other"
)

// ReviewStatus represents the evidence level
type ReviewStatus string

const (
    PracticeGuideline       ReviewStatus = "practice_guideline"
    ReviewedByExpertPanel   ReviewStatus = "reviewed_by_expert_panel"
    CriteriaProvided        ReviewStatus = "criteria_provided"
    MultipleSubmitters      ReviewStatus = "multiple_submitters"
    SingleSubmitter         ReviewStatus = "single_submitter"
    NoAssertion             ReviewStatus = "no_assertion"
)

// VariantType represents the type of genetic variant
type VariantType string

const (
    SNV        VariantType = "SNV"         // Single Nucleotide Variant
    Insertion  VariantType = "insertion"
    Deletion   VariantType = "deletion"
    Indel      VariantType = "indel"
    Duplication VariantType = "duplication"
    CNV        VariantType = "copy_number_variant"
)

// FunctionalClass represents the functional impact
type FunctionalClass string

const (
    Missense        FunctionalClass = "missense"
    Nonsense        FunctionalClass = "nonsense"
    Synonymous      FunctionalClass = "synonymous"
    FrameShift      FunctionalClass = "frameshift"
    Splice          FunctionalClass = "splice"
    UTR5            FunctionalClass = "5_prime_utr"
    UTR3            FunctionalClass = "3_prime_utr"
    Intron          FunctionalClass = "intron"
    Regulatory      FunctionalClass = "regulatory"
    Intergenic      FunctionalClass = "intergenic"
)

// DataSource represents where data came from
type DataSource string

const (
    SourceClinVar   DataSource = "clinvar"
    SourceDbSNP     DataSource = "dbsnp"
    SourceOpenSNP   DataSource = "opensnp"
    SourcePharmGKB  DataSource = "pharmgkb"
    SourceSNPedia   DataSource = "snpedia"
    SourceGnomAD    DataSource = "gnomad"
)

// StringArray is a custom type for storing string slices in SQLite as JSON
type StringArray []string

func (s StringArray) Value() (driver.Value, error) {
    if len(s) == 0 {
        return "[]", nil
    }
    return json.Marshal(s)
}

func (s *StringArray) Scan(value interface{}) error {
    if value == nil {
        *s = []string{}
        return nil
    }
    
    bytes, ok := value.([]byte)
    if !ok {
        return errors.New("failed to scan StringArray")
    }
    
    return json.Unmarshal(bytes, s)
}

// NullableFloat64 for database NULL handling
type NullableFloat64 struct {
    Float64 float64
    Valid   bool
}

func (n NullableFloat64) Value() (driver.Value, error) {
    if !n.Valid {
        return nil, nil
    }
    return n.Float64, nil
}

func (n *NullableFloat64) Scan(value interface{}) error {
    if value == nil {
        n.Float64 = 0
        n.Valid = false
        return nil
    }
    n.Valid = true
    n.Float64 = value.(float64)
    return nil
}
```

### snp.go - Core SNP Model

```go
package models

import (
    "time"
)

// SNP represents a Single Nucleotide Polymorphism
type SNP struct {
    ID                int64       `json:"id" db:"id"`
    RsID              string      `json:"rsid" db:"rsid"`
    Chromosome        string      `json:"chromosome" db:"chromosome"`
    Position          int64       `json:"position" db:"position"`
    ReferenceAllele   string      `json:"reference_allele" db:"reference_allele"`
    AlternateAlleles  StringArray `json:"alternate_alleles" db:"alternate_alleles"`
    GeneSymbol        *string     `json:"gene_symbol,omitempty" db:"gene_symbol"`
    GeneID            *string     `json:"gene_id,omitempty" db:"gene_id"`
    VariantType       VariantType `json:"variant_type" db:"variant_type"`
    FunctionalClass   *FunctionalClass `json:"functional_class,omitempty" db:"functional_class"`
    CreatedAt         time.Time   `json:"created_at" db:"created_at"`
    UpdatedAt         time.Time   `json:"updated_at" db:"updated_at"`
    
    // Related data (not stored in snps table directly)
    Significance      *Significance      `json:"significance,omitempty"`
    ClinicalData      []ClinicalData     `json:"clinical_data,omitempty"`
    Phenotypes        []Phenotype        `json:"phenotypes,omitempty"`
    References        []Reference        `json:"references,omitempty"`
    PopulationData    []PopulationFreq   `json:"population_data,omitempty"`
}

// Validate checks if SNP data is valid
func (s *SNP) Validate() error {
    if s.RsID == "" {
        return errors.New("rsID is required")
    }
    if s.Chromosome == "" {
        return errors.New("chromosome is required")
    }
    if s.Position <= 0 {
        return errors.New("position must be positive")
    }
    if s.ReferenceAllele == "" {
        return errors.New("reference allele is required")
    }
    if len(s.AlternateAlleles) == 0 {
        return errors.New("at least one alternate allele is required")
    }
    return nil
}

// HasGene checks if SNP is associated with a gene
func (s *SNP) HasGene() bool {
    return s.GeneSymbol != nil && *s.GeneSymbol != ""
}

// IsProteinCoding checks if variant affects protein
func (s *SNP) IsProteinCoding() bool {
    if s.FunctionalClass == nil {
        return false
    }
    coding := []FunctionalClass{Missense, Nonsense, FrameShift, Splice}
    for _, class := range coding {
        if *s.FunctionalClass == class {
            return true
        }
    }
    return false
}
```

### significance.go - Significance Scoring

```go
package models

import (
    "encoding/json"
    "time"
)

// Significance represents the calculated significance score
type Significance struct {
    ID              int64              `json:"id" db:"id"`
    SNPID           int64              `json:"snp_id" db:"snp_id"`
    TotalScore      float64            `json:"total_score" db:"total_score"`
    ClinicalScore   float64            `json:"clinical_score" db:"clinical_score"`
    ResearchScore   float64            `json:"research_score" db:"research_score"`
    PopulationScore float64            `json:"population_score" db:"population_score"`
    FunctionalScore float64            `json:"functional_score" db:"functional_score"`
    ScoreDetails    ScoreBreakdown     `json:"score_details" db:"score_details"`
    CalculatedAt    time.Time          `json:"calculated_at" db:"calculated_at"`
}

// ScoreBreakdown contains detailed scoring information
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

// Value implements driver.Valuer for database storage
func (s ScoreBreakdown) Value() (driver.Value, error) {
    return json.Marshal(s)
}

// Scan implements sql.Scanner for database retrieval
func (s *ScoreBreakdown) Scan(value interface{}) error {
    if value == nil {
        return nil
    }
    bytes, ok := value.([]byte)
    if !ok {
        return errors.New("failed to scan ScoreBreakdown")
    }
    return json.Unmarshal(bytes, s)
}

// IsHighlySignificant returns true if score >= 70
func (s *Significance) IsHighlySignificant() bool {
    return s.TotalScore >= 70.0
}

// IsModeratelySignificant returns true if score >= 40
func (s *Significance) IsModeratelySignificant() bool {
    return s.TotalScore >= 40.0
}

// SignificanceLevel returns a human-readable level
func (s *Significance) SignificanceLevel() string {
    switch {
    case s.TotalScore >= 80:
        return "Very High"
    case s.TotalScore >= 60:
        return "High"
    case s.TotalScore >= 40:
        return "Moderate"
    case s.TotalScore >= 20:
        return "Low"
    default:
        return "Minimal"
    }
}
```

### clinical.go - Clinical Data

```go
package models

import (
    "time"
)

// ClinicalData represents clinical significance information
type ClinicalData struct {
    ID                   int64                `json:"id" db:"id"`
    SNPID                int64                `json:"snp_id" db:"snp_id"`
    ClinicalSignificance ClinicalSignificance `json:"clinical_significance" db:"clinical_significance"`
    ReviewStatus         ReviewStatus         `json:"review_status" db:"review_status"`
    ConditionName        string               `json:"condition_name" db:"condition_name"`
    ConditionID          *string              `json:"condition_id,omitempty" db:"condition_id"`
    InheritancePattern   *string              `json:"inheritance_pattern,omitempty" db:"inheritance_pattern"`
    Penetrance           *string              `json:"penetrance,omitempty" db:"penetrance"`
    AlleleOrigin         *string              `json:"allele_origin,omitempty" db:"allele_origin"`
    Source               DataSource           `json:"source" db:"source"`
    SourceID             *string              `json:"source_id,omitempty" db:"source_id"`
    LastEvaluated        *time.Time           `json:"last_evaluated,omitempty" db:"last_evaluated"`
    CreatedAt            time.Time            `json:"created_at" db:"created_at"`
}

// IsPathogenic returns true if variant is pathogenic or likely pathogenic
func (c *ClinicalData) IsPathogenic() bool {
    return c.ClinicalSignificance == Pathogenic ||
           c.ClinicalSignificance == LikelyPathogenic
}

// IsBenign returns true if variant is benign or likely benign
func (c *ClinicalData) IsBenign() bool {
    return c.ClinicalSignificance == Benign ||
           c.ClinicalSignificance == LikelyBenign
}

// HasHighEvidence returns true if reviewed by experts or practice guideline
func (c *ClinicalData) HasHighEvidence() bool {
    return c.ReviewStatus == PracticeGuideline ||
           c.ReviewStatus == ReviewedByExpertPanel
}
```

### phenotype.go - Phenotype Associations

```go
package models

import (
    "time"
)

// Phenotype represents a trait or condition associated with SNP
type Phenotype struct {
    ID                int64              `json:"id" db:"id"`
    SNPID             int64              `json:"snp_id" db:"snp_id"`
    PhenotypeName     string             `json:"phenotype_name" db:"phenotype_name"`
    PhenotypeID       *string            `json:"phenotype_id,omitempty" db:"phenotype_id"`
    AssociationType   string             `json:"association_type" db:"association_type"`
    OddsRatio         *NullableFloat64   `json:"odds_ratio,omitempty" db:"odds_ratio"`
    ConfidenceInterval *string           `json:"confidence_interval,omitempty" db:"confidence_interval"`
    PValue            *NullableFloat64   `json:"p_value,omitempty" db:"p_value"`
    StudyType         *string            `json:"study_type,omitempty" db:"study_type"`
    Source            DataSource         `json:"source" db:"source"`
    CreatedAt         time.Time          `json:"created_at" db:"created_at"`
}

// IsStatisticallySignificant checks if p-value < 0.05
func (p *Phenotype) IsStatisticallySignificant() bool {
    if p.PValue == nil || !p.PValue.Valid {
        return false
    }
    return p.PValue.Float64 < 0.05
}

// HasStrongEffect checks if odds ratio > 2 or < 0.5
func (p *Phenotype) HasStrongEffect() bool {
    if p.OddsRatio == nil || !p.OddsRatio.Valid {
        return false
    }
    return p.OddsRatio.Float64 > 2.0 || p.OddsRatio.Float64 < 0.5
}
```

### reference.go - Research References

```go
package models

import (
    "time"
)

// Reference represents a research paper or study
type Reference struct {
    ID              int64     `json:"id" db:"id"`
    SNPID           int64     `json:"snp_id" db:"snp_id"`
    PubmedID        *string   `json:"pubmed_id,omitempty" db:"pubmed_id"`
    Title           *string   `json:"title,omitempty" db:"title"`
    Authors         *string   `json:"authors,omitempty" db:"authors"`
    Journal         *string   `json:"journal,omitempty" db:"journal"`
    PublicationYear *int      `json:"publication_year,omitempty" db:"publication_year"`
    DOI             *string   `json:"doi,omitempty" db:"doi"`
    URL             *string   `json:"url,omitempty" db:"url"`
    CitationCount   int       `json:"citation_count" db:"citation_count"`
    Abstract        *string   `json:"abstract,omitempty" db:"abstract"`
    CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// GetPubmedURL returns the full PubMed URL
func (r *Reference) GetPubmedURL() string {
    if r.PubmedID == nil {
        return ""
    }
    return "https://pubmed.ncbi.nlm.nih.gov/" + *r.PubmedID
}

// IsHighlyCited returns true if citation count > 100
func (r *Reference) IsHighlyCited() bool {
    return r.CitationCount > 100
}
```

### population.go - Population Frequencies

```go
package models

import (
    "time"
)

// PopulationFreq represents allele frequency in a population
type PopulationFreq struct {
    ID              int64      `json:"id" db:"id"`
    SNPID           int64      `json:"snp_id" db:"snp_id"`
    PopulationCode  string     `json:"population_code" db:"population_code"`
    PopulationName  *string    `json:"population_name,omitempty" db:"population_name"`
    Allele          string     `json:"allele" db:"allele"`
    Frequency       float64    `json:"frequency" db:"frequency"`
    AlleleCount     *int       `json:"allele_count,omitempty" db:"allele_count"`
    AlleleNumber    *int       `json:"allele_number,omitempty" db:"allele_number"`
    HomozygoteCount *int       `json:"homozygote_count,omitempty" db:"homozygote_count"`
    Source          DataSource `json:"source" db:"source"`
    CreatedAt       time.Time  `json:"created_at" db:"created_at"`
}

// IsCommon returns true if frequency > 5%
func (p *PopulationFreq) IsCommon() bool {
    return p.Frequency > 0.05
}

// IsRare returns true if frequency < 1%
func (p *PopulationFreq) IsRare() bool {
    return p.Frequency < 0.01
}
```

### translation.go - Translation Support

```go
package models

import (
    "time"
)

// Translation represents translated content
type Translation struct {
    ID             int64     `json:"id" db:"id"`
    SNPID          int64     `json:"snp_id" db:"snp_id"`
    LanguageCode   string    `json:"language_code" db:"language_code"`
    FieldName      string    `json:"field_name" db:"field_name"`
    TranslatedText string    `json:"translated_text" db:"translated_text"`
    Translator     *string   `json:"translator,omitempty" db:"translator"`
    TranslatedAt   time.Time `json:"translated_at" db:"translated_at"`
    Verified       bool      `json:"verified" db:"verified"`
}

// PhenotypeTranslation represents translated phenotype names
type PhenotypeTranslation struct {
    ID             int64     `json:"id" db:"id"`
    PhenotypeID    int64     `json:"phenotype_id" db:"phenotype_id"`
    LanguageCode   string    `json:"language_code" db:"language_code"`
    TranslatedName string    `json:"translated_name" db:"translated_name"`
    Translator     *string   `json:"translator,omitempty" db:"translator"`
    TranslatedAt   time.Time `json:"translated_at" db:"translated_at"`
    Verified       bool      `json:"verified" db:"verified"`
}
```

### source.go - Data Source Metadata

```go
package models

import (
    "time"
)

// SourceMetadata represents information about a data source
type SourceMetadata struct {
    ID           int64     `json:"id" db:"id"`
    SourceName   string    `json:"source_name" db:"source_name"`
    SourceURL    string    `json:"source_url" db:"source_url"`
    APIVersion   *string   `json:"api_version,omitempty" db:"api_version"`
    Description  *string   `json:"description,omitempty" db:"description"`
    TermsOfUse   *string   `json:"terms_of_use,omitempty" db:"terms_of_use"`
    LastAccessed *time.Time `json:"last_accessed,omitempty" db:"last_accessed"`
    IsActive     bool      `json:"is_active" db:"is_active"`
    CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// DownloadMetadata tracks download runs
type DownloadMetadata struct {
    ID             int64     `json:"id" db:"id"`
    RunID          string    `json:"run_id" db:"run_id"`
    Source         string    `json:"source" db:"source"`
    StartTime      time.Time `json:"start_time" db:"start_time"`
    EndTime        *time.Time `json:"end_time,omitempty" db:"end_time"`
    Status         string    `json:"status" db:"status"`
    SNPsDownloaded int       `json:"snps_downloaded" db:"snps_downloaded"`
    SNPsUpdated    int       `json:"snps_updated" db:"snps_updated"`
    SNPsSkipped    int       `json:"snps_skipped" db:"snps_skipped"`
    ErrorsCount    int       `json:"errors_count" db:"errors_count"`
    ErrorLog       *string   `json:"error_log,omitempty" db:"error_log"`
    ConfigSnapshot *string   `json:"config_snapshot,omitempty" db:"config_snapshot"`
    CreatedAt      time.Time `json:"created_at" db:"created_at"`
}
```

## Implementation Tasks

1. Create all model files in `internal/models/`
2. Add JSON tags for API responses
3. Add db tags for database mapping
4. Implement validation methods
5. Add helper methods for common queries
6. Write unit tests for each model
7. Document all exported types and methods

## Testing Strategy

```go
func TestSNPValidation(t *testing.T) {
    validSNP := &SNP{
        RsID: "rs429358",
        Chromosome: "19",
        Position: 44908684,
        ReferenceAllele: "C",
        AlternateAlleles: []string{"T"},
    }
    
    if err := validSNP.Validate(); err != nil {
        t.Errorf("Valid SNP failed validation: %v", err)
    }
}
```

## Dependencies

```go
// go.mod additions
require (
    github.com/google/uuid v1.5.0  // for run IDs
)
```

## Success Criteria
- All models defined with proper types
- JSON marshaling works correctly
- Database scanning works correctly
- Validation catches invalid data
- Helper methods are intuitive
- 100% test coverage on validation logic

## Next Feature
After completing this, proceed to **003 - Rate Limiting System**.
