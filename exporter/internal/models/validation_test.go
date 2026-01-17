package models

import "testing"

func TestSNPValidate(t *testing.T) {
	valid := &SNP{
		RsID:            "rs429358",
		Chromosome:      "19",
		Position:        44908684,
		ReferenceAllele: "C",
		AlternateAlleles: StringArray{
			"T",
		},
		VariantType: VariantSNV,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid SNP, got error: %v", err)
	}

	invalid := &SNP{}
	if err := invalid.Validate(); err == nil {
		t.Fatalf("expected error for invalid SNP")
	}
}

func TestPhenotypeChecks(t *testing.T) {
	p := &Phenotype{}
	if p.IsStatisticallySignificant() {
		t.Fatalf("expected insignificant when p-value missing")
	}
	if p.HasStrongEffect() {
		t.Fatalf("expected no strong effect when odds ratio missing")
	}

	p.PValue = &NullableFloat64{Float64: 0.01, Valid: true}
	if !p.IsStatisticallySignificant() {
		t.Fatalf("expected significant when p-value < 0.05")
	}

	p.OddsRatio = &NullableFloat64{Float64: 3.0, Valid: true}
	if !p.HasStrongEffect() {
		t.Fatalf("expected strong effect when odds ratio > 2")
	}
}

func TestClinicalHelpers(t *testing.T) {
	c := &ClinicalData{ClinicalSignificance: ClinicalPathogenic, ReviewStatus: ReviewExpertPanel}
	if !c.IsPathogenic() {
		t.Fatalf("expected pathogenic")
	}
	if !c.HasHighEvidence() {
		t.Fatalf("expected high evidence")
	}
	c.ClinicalSignificance = ClinicalBenign
	if !c.IsBenign() {
		t.Fatalf("expected benign")
	}
}

func TestSignificanceHelpers(t *testing.T) {
	s := &Significance{TotalScore: 85}
	if !s.IsHighlySignificant() {
		t.Fatalf("expected highly significant")
	}
	s.TotalScore = 45
	if !s.IsModeratelySignificant() {
		t.Fatalf("expected moderately significant")
	}
	s.TotalScore = 10
	if lvl := s.SignificanceLevel(); lvl != "Minimal" {
		t.Fatalf("expected Minimal, got %s", lvl)
	}
}

func TestReferenceHelpers(t *testing.T) {
	r := &Reference{CitationCount: 101}
	if !r.IsHighlyCited() {
		t.Fatalf("expected highly cited")
	}
	pmid := "123"
	r.PubmedID = &pmid
	if got := r.GetPubmedURL(); got != "https://pubmed.ncbi.nlm.nih.gov/123" {
		t.Fatalf("unexpected pubmed url: %s", got)
	}
}

func TestPopulationHelpers(t *testing.T) {
	pop := &PopulationFreq{Frequency: 0.06}
	if !pop.IsCommon() {
		t.Fatalf("expected common")
	}
	pop.Frequency = 0.005
	if !pop.IsRare() {
		t.Fatalf("expected rare")
	}
}
