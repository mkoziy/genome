package clinvar

import (
	"fmt"
	"strings"
	"time"

	"github.com/mkoziy/genome/exporter/internal/models"
)

// MapToSNP converts ClinVarSet to SNP model.
func MapToSNP(cvSet ClinVarSet) (*models.SNP, error) {
	ref := cvSet.ReferenceClinVarAssertion
	if len(ref.MeasureSet.Measure) == 0 {
		return nil, fmt.Errorf("no measures in variant")
	}
	measure := ref.MeasureSet.Measure[0]

	rsID := extractRsID(measure.XRef)
	if rsID == "" {
		return nil, fmt.Errorf("no rsID found")
	}

	seqLoc := findGRCh38Location(measure.SequenceLocation)
	if seqLoc == nil {
		return nil, fmt.Errorf("no GRCh38 location found")
	}

	geneSymbol := extractGeneSymbol(measure.MeasureRelationship)

	varType := models.VariantSNV
	if measure.Type != "" {
		varType = models.VariantType(measure.Type)
	}

	funcClass := extractFunctionalClass(measure.AttributeSet)

	snp := &models.SNP{
		RsID:             rsID,
		Chromosome:       seqLoc.Chr,
		Position:         seqLoc.Start,
		ReferenceAllele:  seqLoc.ReferenceAllele,
		AlternateAlleles: models.StringArray{seqLoc.AlternateAllele},
		GeneSymbol:       geneSymbol,
		VariantType:      varType,
		FunctionalClass:  funcClass,
	}
	return snp, nil
}

// MapToClinical converts clinical significance data.
func MapToClinical(cvSet ClinVarSet, snpID int64) []models.ClinicalData {
	ref := cvSet.ReferenceClinVarAssertion
	clinSig := ref.ClinicalSignificance

	result := make([]models.ClinicalData, 0)
	for _, trait := range ref.TraitSet.Trait {
		conditionName := extractConditionName(trait.Name)
		if conditionName == "" {
			continue
		}
		conditionID := extractConditionID(trait.XRef)
		lastEval := parseDate(clinSig.DateLastEvaluated)

		clinical := models.ClinicalData{
			SNPID:                snpID,
			ClinicalSignificance: mapClinicalSignificance(clinSig.Description),
			ReviewStatus:         mapReviewStatus(clinSig.ReviewStatus),
			ConditionName:        conditionName,
			ConditionID:          conditionID,
			Source:               models.SourceClinVar,
			SourceID:             &cvSet.ReferenceClinVarAssertion.ClinVarAccession.Acc,
			LastEvaluated:        lastEval,
		}
		result = append(result, clinical)
	}
	return result
}

// MapToReferences extracts PubMed references.
func MapToReferences(cvSet ClinVarSet, snpID int64) []models.Reference {
	refs := make([]models.Reference, 0)
	seen := make(map[string]bool)

	for _, citation := range cvSet.ReferenceClinVarAssertion.ClinicalSignificance.Citation {
		pmid := extractPubMedID(citation.ID)
		if pmid != "" && !seen[pmid] {
			refs = append(refs, models.Reference{SNPID: snpID, PubmedID: &pmid})
			seen[pmid] = true
		}
	}

	for _, obs := range cvSet.ReferenceClinVarAssertion.ObservedIn {
		for _, obsData := range obs.ObservedData {
			for _, citation := range obsData.Citation {
				pmid := extractPubMedID(citation.ID)
				if pmid != "" && !seen[pmid] {
					refs = append(refs, models.Reference{SNPID: snpID, PubmedID: &pmid})
					seen[pmid] = true
				}
			}
		}
	}
	return refs
}

// Helpers

func extractRsID(xrefs []XRef) string {
	for _, xref := range xrefs {
		if xref.DB == "dbSNP" && strings.HasPrefix(xref.ID, "rs") {
			return xref.ID
		}
	}
	return ""
}

func findGRCh38Location(locs []SequenceLocation) *SequenceLocation {
	for _, loc := range locs {
		if loc.Assembly == "GRCh38" {
			return &loc
		}
	}
	return nil
}

func extractGeneSymbol(rels []MeasureRelationship) *string {
	for _, rel := range rels {
		if rel.Type == "genes overlapped by variant" && len(rel.Symbol) > 0 {
			symbol := rel.Symbol[0].ElementValue.Value
			return &symbol
		}
	}
	return nil
}

func extractFunctionalClass(attrs []AttributeSet) *models.FunctionalClass {
	for _, attr := range attrs {
		if attr.Attribute.Type == "MolecularConsequence" {
			class := models.FunctionalClass(strings.ToLower(attr.Attribute.Value))
			return &class
		}
	}
	return nil
}

func extractConditionName(names []Name) string {
	for _, name := range names {
		if name.ElementValue.Type == "Preferred" {
			return name.ElementValue.Value
		}
	}
	if len(names) > 0 {
		return names[0].ElementValue.Value
	}
	return ""
}

func extractConditionID(xrefs []XRef) *string {
	for _, xref := range xrefs {
		if xref.DB == "MedGen" {
			return &xref.ID
		}
	}
	return nil
}

func extractPubMedID(ids []ID) string {
	for _, id := range ids {
		if id.Source == "PubMed" {
			return id.Value
		}
	}
	return ""
}

func mapClinicalSignificance(desc string) models.ClinicalSignificance {
	desc = strings.ToLower(strings.TrimSpace(desc))
	switch {
	case strings.Contains(desc, "pathogenic") && strings.Contains(desc, "likely"):
		return models.ClinicalLikelyPathogenic
	case strings.Contains(desc, "pathogenic"):
		return models.ClinicalPathogenic
	case strings.Contains(desc, "benign") && strings.Contains(desc, "likely"):
		return models.ClinicalLikelyBenign
	case strings.Contains(desc, "benign"):
		return models.ClinicalBenign
	case strings.Contains(desc, "risk"):
		return models.ClinicalRiskFactor
	case strings.Contains(desc, "protective"):
		return models.ClinicalProtective
	case strings.Contains(desc, "drug"):
		return models.ClinicalDrugResponse
	case strings.Contains(desc, "uncertain"):
		return models.ClinicalUncertainSignif
	default:
		return models.ClinicalOther
	}
}

func mapReviewStatus(status string) models.ReviewStatus {
	status = strings.ToLower(strings.TrimSpace(status))
	switch {
	case strings.Contains(status, "practice guideline"):
		return models.ReviewPracticeGuideline
	case strings.Contains(status, "expert panel"):
		return models.ReviewExpertPanel
	case strings.Contains(status, "criteria provided"):
		return models.ReviewCriteriaProvided
	case strings.Contains(status, "multiple"):
		return models.ReviewMultipleSubmitter
	case strings.Contains(status, "single"):
		return models.ReviewSingleSubmitter
	default:
		return models.ReviewNoAssertion
	}
}

func parseDate(dateStr string) *time.Time {
	if dateStr == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil
	}
	return &t
}
