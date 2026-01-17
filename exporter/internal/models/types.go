package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// ClinicalSignificance represents the clinical impact.
type ClinicalSignificance string

const (
	ClinicalPathogenic       ClinicalSignificance = "pathogenic"
	ClinicalLikelyPathogenic ClinicalSignificance = "likely_pathogenic"
	ClinicalUncertainSignif  ClinicalSignificance = "uncertain_significance"
	ClinicalLikelyBenign     ClinicalSignificance = "likely_benign"
	ClinicalBenign           ClinicalSignificance = "benign"
	ClinicalRiskFactor       ClinicalSignificance = "risk_factor"
	ClinicalProtective       ClinicalSignificance = "protective"
	ClinicalDrugResponse     ClinicalSignificance = "drug_response"
	ClinicalAssociation      ClinicalSignificance = "association"
	ClinicalOther            ClinicalSignificance = "other"
)

// Review status per ClinVar.
type ReviewStatus string

const (
	ReviewPracticeGuideline ReviewStatus = "practice_guideline"
	ReviewExpertPanel       ReviewStatus = "reviewed_by_expert_panel"
	ReviewCriteriaProvided  ReviewStatus = "criteria_provided"
	ReviewMultipleSubmitter ReviewStatus = "multiple_submitters"
	ReviewSingleSubmitter   ReviewStatus = "single_submitter"
	ReviewNoAssertion       ReviewStatus = "no_assertion"
)

// Data source tagging to track provenance.
type DataSource string

const (
	SourceClinVar  DataSource = "clinvar"
	SourceDbSNP    DataSource = "dbsnp"
	SourceOpenSNP  DataSource = "opensnp"
	SourcePharmGKB DataSource = "pharmgkb"
	SourceSNPedia  DataSource = "snpedia"
	SourceGnomAD   DataSource = "gnomad"
)

// Variant type for SNP characterization.
type VariantType string

const (
	VariantSNV         VariantType = "SNV"
	VariantInsertion   VariantType = "insertion"
	VariantDeletion    VariantType = "deletion"
	VariantIndel       VariantType = "indel"
	VariantDuplication VariantType = "duplication"
	VariantCNV         VariantType = "copy_number_variant"
)

// Functional class approximations.
type FunctionalClass string

const (
	FuncMissense   FunctionalClass = "missense"
	FuncNonsense   FunctionalClass = "nonsense"
	FuncSynonymous FunctionalClass = "synonymous"
	FuncFrameShift FunctionalClass = "frameshift"
	FuncSplice     FunctionalClass = "splice"
	FuncUTR5       FunctionalClass = "5_prime_utr"
	FuncUTR3       FunctionalClass = "3_prime_utr"
	FuncIntron     FunctionalClass = "intron"
	FuncRegulatory FunctionalClass = "regulatory"
	FuncIntergenic FunctionalClass = "intergenic"
)

// StringArray stores a slice of strings in SQLite as JSON.
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

// NullableFloat64 handles nullable float columns.
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

	switch v := value.(type) {
	case float64:
		n.Float64 = v
	case []byte:
		if err := json.Unmarshal(v, &n.Float64); err != nil {
			return err
		}
	default:
		return errors.New("failed to scan NullableFloat64")
	}

	n.Valid = true
	return nil
}
