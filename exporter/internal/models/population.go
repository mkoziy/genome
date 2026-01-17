package models

import (
	"time"

	"github.com/uptrace/bun"
)

// PopulationFreq represents allele frequency in a population for a SNP.
type PopulationFreq struct {
	bun.BaseModel `bun:"table:snp_populations,alias:pop"`

	ID              int64      `bun:"id,pk,autoincrement" json:"id"`
	SNPID           int64      `bun:"snp_id,notnull" json:"snp_id"`
	PopulationCode  string     `bun:"population_code,notnull" json:"population_code"`
	PopulationName  *string    `bun:"population_name" json:"population_name,omitempty"`
	Allele          string     `bun:"allele,notnull" json:"allele"`
	Frequency       float64    `bun:"frequency,notnull" json:"frequency"`
	AlleleCount     *int       `bun:"allele_count" json:"allele_count,omitempty"`
	AlleleNumber    *int       `bun:"allele_number" json:"allele_number,omitempty"`
	HomozygoteCount *int       `bun:"homozygote_count" json:"homozygote_count,omitempty"`
	Source          DataSource `bun:"source,notnull" json:"source"`
	CreatedAt       time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`

	SNP *SNP `bun:"rel:belongs-to,join:snp_id=id" json:"-"`
}

// IsCommon returns true if frequency > 5%.
func (p *PopulationFreq) IsCommon() bool {
	return p.Frequency > 0.05
}

// IsRare returns true if frequency < 1%.
func (p *PopulationFreq) IsRare() bool {
	return p.Frequency < 0.01
}
