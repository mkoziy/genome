package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Phenotype represents a trait or condition associated with a SNP.
type Phenotype struct {
	bun.BaseModel `bun:"table:snp_phenotypes,alias:p"`

	ID                 int64            `bun:"id,pk,autoincrement" json:"id"`
	SNPID              int64            `bun:"snp_id,notnull" json:"snp_id"`
	PhenotypeName      string           `bun:"phenotype_name,notnull" json:"phenotype_name"`
	PhenotypeID        *string          `bun:"phenotype_id" json:"phenotype_id,omitempty"`
	AssociationType    string           `bun:"association_type,notnull" json:"association_type"`
	OddsRatio          *NullableFloat64 `bun:"odds_ratio" json:"odds_ratio,omitempty"`
	ConfidenceInterval *string          `bun:"confidence_interval" json:"confidence_interval,omitempty"`
	PValue             *NullableFloat64 `bun:"p_value" json:"p_value,omitempty"`
	StudyType          *string          `bun:"study_type" json:"study_type,omitempty"`
	Source             DataSource       `bun:"source,notnull" json:"source"`
	CreatedAt          time.Time        `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`

	SNP *SNP `bun:"rel:belongs-to,join:snp_id=id" json:"-"`
}

// IsStatisticallySignificant checks if p-value < 0.05.
func (p *Phenotype) IsStatisticallySignificant() bool {
	if p.PValue == nil || !p.PValue.Valid {
		return false
	}
	return p.PValue.Float64 < 0.05
}

// HasStrongEffect checks if odds ratio > 2 or < 0.5.
func (p *Phenotype) HasStrongEffect() bool {
	if p.OddsRatio == nil || !p.OddsRatio.Valid {
		return false
	}
	return p.OddsRatio.Float64 > 2.0 || p.OddsRatio.Float64 < 0.5
}
