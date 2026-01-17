package models

import (
	"time"

	"github.com/uptrace/bun"
)

// ClinicalData represents clinical significance annotations for a SNP.
type ClinicalData struct {
	bun.BaseModel `bun:"table:snp_clinical,alias:c"`

	ID                   int64                `bun:"id,pk,autoincrement" json:"id"`
	SNPID                int64                `bun:"snp_id,notnull" json:"snp_id"`
	ClinicalSignificance ClinicalSignificance `bun:"clinical_significance,notnull" json:"clinical_significance"`
	ReviewStatus         ReviewStatus         `bun:"review_status,notnull" json:"review_status"`
	ConditionName        string               `bun:"condition_name,notnull" json:"condition_name"`
	ConditionID          *string              `bun:"condition_id" json:"condition_id,omitempty"`
	InheritancePattern   *string              `bun:"inheritance_pattern" json:"inheritance_pattern,omitempty"`
	Penetrance           *string              `bun:"penetrance" json:"penetrance,omitempty"`
	AlleleOrigin         *string              `bun:"allele_origin" json:"allele_origin,omitempty"`
	Source               DataSource           `bun:"source,notnull" json:"source"`
	SourceID             *string              `bun:"source_id" json:"source_id,omitempty"`
	LastEvaluated        *time.Time           `bun:"last_evaluated" json:"last_evaluated,omitempty"`
	CreatedAt            time.Time            `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`

	SNP *SNP `bun:"rel:belongs-to,join:snp_id=id" json:"-"`
}

// IsPathogenic returns true if variant is pathogenic or likely pathogenic.
func (c *ClinicalData) IsPathogenic() bool {
	return c.ClinicalSignificance == ClinicalPathogenic || c.ClinicalSignificance == ClinicalLikelyPathogenic
}

// IsBenign returns true if variant is benign or likely benign.
func (c *ClinicalData) IsBenign() bool {
	return c.ClinicalSignificance == ClinicalBenign || c.ClinicalSignificance == ClinicalLikelyBenign
}

// HasHighEvidence returns true if reviewed by experts or practice guideline.
func (c *ClinicalData) HasHighEvidence() bool {
	return c.ReviewStatus == ReviewPracticeGuideline || c.ReviewStatus == ReviewExpertPanel
}
