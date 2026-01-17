package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/uptrace/bun"
)

// Significance represents the calculated significance score for a SNP.
type Significance struct {
	bun.BaseModel `bun:"table:snp_significance,alias:sig"`

	ID              int64          `bun:"id,pk,autoincrement" json:"id"`
	SNPID           int64          `bun:"snp_id,notnull,unique" json:"snp_id"`
	TotalScore      float64        `bun:"total_score,notnull" json:"total_score"`
	ClinicalScore   float64        `bun:"clinical_score,notnull" json:"clinical_score"`
	ResearchScore   float64        `bun:"research_score,notnull" json:"research_score"`
	PopulationScore float64        `bun:"population_score,notnull" json:"population_score"`
	FunctionalScore float64        `bun:"functional_score,notnull" json:"functional_score"`
	ScoreDetails    ScoreBreakdown `bun:"score_details,type:json" json:"score_details"`
	CalculatedAt    time.Time      `bun:"calculated_at,nullzero,notnull,default:current_timestamp" json:"calculated_at"`

	SNP *SNP `bun:"rel:belongs-to,join:snp_id=id" json:"-"`
}

// ScoreBreakdown stores per-dimension scores.
type ScoreBreakdown struct {
	ClinicalDetails   ClinicalScoring   `json:"clinical"`
	ResearchDetails   ResearchScoring   `json:"research"`
	PopulationDetails PopulationScoring `json:"population"`
	FunctionalDetails FunctionalScoring `json:"functional"`
}

func (s ScoreBreakdown) Value() (driver.Value, error) {
	return json.Marshal(s)
}

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

type ClinicalScoring struct {
	HasPathogenic     bool    `json:"has_pathogenic"`
	ReviewStatusScore float64 `json:"review_status_score"`
	ConditionCount    int     `json:"condition_count"`
}

type ResearchScoring struct {
	PubmedCount       int `json:"pubmed_count"`
	CitationTotal     int `json:"citation_total"`
	HighImpactStudies int `json:"high_impact_studies"`
}

type PopulationScoring struct {
	MaxMAF          float64 `json:"max_maf"`
	PopulationCount int     `json:"population_count"`
}

type FunctionalScoring struct {
	IsProteinChanging bool `json:"is_protein_changing"`
	IsRegulatory      bool `json:"is_regulatory"`
}

// IsHighlySignificant returns true if score >= 70.
func (s *Significance) IsHighlySignificant() bool {
	return s.TotalScore >= 70.0
}

// IsModeratelySignificant returns true if score >= 40.
func (s *Significance) IsModeratelySignificant() bool {
	return s.TotalScore >= 40.0
}

// SignificanceLevel returns a human-readable level.
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
