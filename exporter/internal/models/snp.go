package models

import (
	"context"
	"errors"
	"time"

	"github.com/uptrace/bun"
)

// SNP represents a Single Nucleotide Polymorphism.
type SNP struct {
	bun.BaseModel `bun:"table:snps,alias:s"`

	ID               int64            `bun:"id,pk,autoincrement" json:"id"`
	RsID             string           `bun:"rsid,unique,notnull" json:"rsid"`
	Chromosome       string           `bun:"chromosome,notnull" json:"chromosome"`
	Position         int64            `bun:"position,notnull" json:"position"`
	ReferenceAllele  string           `bun:"reference_allele,notnull" json:"reference_allele"`
	AlternateAlleles StringArray      `bun:"alternate_alleles,type:json,notnull" json:"alternate_alleles"`
	GeneSymbol       *string          `bun:"gene_symbol" json:"gene_symbol,omitempty"`
	GeneID           *string          `bun:"gene_id" json:"gene_id,omitempty"`
	VariantType      VariantType      `bun:"variant_type,notnull" json:"variant_type"`
	FunctionalClass  *FunctionalClass `bun:"functional_class" json:"functional_class,omitempty"`
	CreatedAt        time.Time        `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt        time.Time        `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updated_at"`

	Significance   *Significance     `bun:"rel:has-one,join:id=snp_id" json:"significance,omitempty"`
	ClinicalData   []*ClinicalData   `bun:"rel:has-many,join:id=snp_id" json:"clinical_data,omitempty"`
	Phenotypes     []*Phenotype      `bun:"rel:has-many,join:id=snp_id" json:"phenotypes,omitempty"`
	References     []*Reference      `bun:"rel:has-many,join:id=snp_id" json:"references,omitempty"`
	PopulationData []*PopulationFreq `bun:"rel:has-many,join:id=snp_id" json:"population_data,omitempty"`
}

// BeforeUpdate updates the timestamp on modifications.
func (s *SNP) BeforeUpdate(ctx context.Context, query *bun.UpdateQuery) error {
	s.UpdatedAt = time.Now()
	return nil
}

// Validate checks that required SNP fields are present.
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

// HasGene reports whether the SNP is associated with a gene.
func (s *SNP) HasGene() bool {
	return s.GeneSymbol != nil && *s.GeneSymbol != ""
}

// IsProteinCoding checks if the variant likely affects protein.
func (s *SNP) IsProteinCoding() bool {
	if s.FunctionalClass == nil {
		return false
	}
	coding := []FunctionalClass{FuncMissense, FuncNonsense, FuncFrameShift, FuncSplice}
	for _, class := range coding {
		if *s.FunctionalClass == class {
			return true
		}
	}
	return false
}
