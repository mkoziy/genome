package clinvar

import (
	"fmt"
	"strings"
)

// QueryBuilder builds ClinVar search queries.
type QueryBuilder struct {
	terms []string
}

// NewQueryBuilder creates a new query builder.
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{terms: make([]string, 0)}
}

// WithClinicalSignificance adds clinical significance filter.
func (qb *QueryBuilder) WithClinicalSignificance(significance ...string) *QueryBuilder {
	if len(significance) > 0 {
		sigTerms := make([]string, len(significance))
		for i, sig := range significance {
			sigTerms[i] = fmt.Sprintf("%s[CLNSIG]", sig)
		}
		qb.terms = append(qb.terms, fmt.Sprintf("(%s)", strings.Join(sigTerms, " OR ")))
	}
	return qb
}

// WithReviewStatus adds review status filter.
func (qb *QueryBuilder) WithReviewStatus(status ...string) *QueryBuilder {
	if len(status) > 0 {
		statusTerms := make([]string, len(status))
		for i, s := range status {
			statusTerms[i] = fmt.Sprintf("\"%s\"[RVSTAT]", s)
		}
		qb.terms = append(qb.terms, fmt.Sprintf("(%s)", strings.Join(statusTerms, " OR ")))
	}
	return qb
}

// WithVariantType adds variant type filter.
func (qb *QueryBuilder) WithVariantType(varType string) *QueryBuilder {
	if varType != "" {
		qb.terms = append(qb.terms, fmt.Sprintf("%s[VARTYPE]", varType))
	}
	return qb
}

// WithGene adds gene symbol filter.
func (qb *QueryBuilder) WithGene(gene string) *QueryBuilder {
	if gene != "" {
		qb.terms = append(qb.terms, fmt.Sprintf("%s[GENE]", gene))
	}
	return qb
}

// Build constructs the final query string.
func (qb *QueryBuilder) Build() string {
	if len(qb.terms) == 0 {
		return ""
	}
	return strings.Join(qb.terms, " AND ")
}

// Predefined queries.
func QueryPathogenicVariants() string {
	return NewQueryBuilder().
		WithClinicalSignificance("pathogenic", "likely pathogenic").
		Build()
}

func QueryRiskFactorVariants() string {
	return NewQueryBuilder().
		WithClinicalSignificance("risk factor", "affects").
		Build()
}

func QueryDrugResponseVariants() string {
	return NewQueryBuilder().
		WithClinicalSignificance("drug response").
		Build()
}

func QueryHighEvidenceVariants() string {
	return NewQueryBuilder().
		WithClinicalSignificance("pathogenic", "likely pathogenic", "risk factor").
		WithReviewStatus("practice guideline", "reviewed by expert panel").
		Build()
}
