package clinvar

import "encoding/xml"

// SearchResponse represents ESearch response
type SearchResponse struct {
	Count    string   `json:"count"`
	RetMax   string   `json:"retmax"`
	RetStart string   `json:"retstart"`
	IdList   []string `json:"idlist"`
	WebEnv   string   `json:"webenv"`
	QueryKey string   `json:"querykey"`
}

// ClinVarSet is the root element for variant details
type ClinVarSet struct {
	XMLName                   xml.Name                  `xml:"ClinVarSet"`
	ReferenceClinVarAssertion ReferenceClinVarAssertion `xml:"ReferenceClinVarAssertion"`
}

// ReferenceClinVarAssertion contains the main variant information
type ReferenceClinVarAssertion struct {
	ClinVarAccession     ClinVarAccession     `xml:"ClinVarAccession"`
	ClinicalSignificance ClinicalSignificance `xml:"ClinicalSignificance"`
	MeasureSet           MeasureSet           `xml:"MeasureSet"`
	TraitSet             TraitSet             `xml:"TraitSet"`
	ObservedIn           []ObservedIn         `xml:"ObservedIn"`
}

// ClinVarAccession contains the variant ID
type ClinVarAccession struct {
	Acc     string `xml:"Acc,attr"`
	Version int    `xml:"Version,attr"`
	Type    string `xml:"Type,attr"`
}

// ClinicalSignificance contains the clinical interpretation
type ClinicalSignificance struct {
	ReviewStatus      string     `xml:"ReviewStatus"`
	Description       string     `xml:"Description"`
	DateLastEvaluated string     `xml:"DateLastEvaluated,attr"`
	Citation          []Citation `xml:"Citation"`
}

// MeasureSet contains one or more variants
type MeasureSet struct {
	Type    string    `xml:"Type,attr"`
	Measure []Measure `xml:"Measure"`
}

// Measure contains variant details
type Measure struct {
	Type                string                `xml:"Type,attr"`
	Name                []Name                `xml:"Name"`
	AttributeSet        []AttributeSet        `xml:"AttributeSet"`
	MeasureRelationship []MeasureRelationship `xml:"MeasureRelationship"`
	SequenceLocation    []SequenceLocation    `xml:"SequenceLocation"`
	XRef                []XRef                `xml:"XRef"`
}

// Name contains variant naming information
type Name struct {
	ElementValue ElementValue `xml:"ElementValue"`
}

// ElementValue contains the actual name text
type ElementValue struct {
	Type  string `xml:"Type,attr"`
	Value string `xml:",chardata"`
}

// AttributeSet contains various attributes
type AttributeSet struct {
	Attribute Attribute `xml:"Attribute"`
}

// Attribute contains key-value pairs
type Attribute struct {
	Type  string `xml:"Type,attr"`
	Value string `xml:",chardata"`
}

// MeasureRelationship links to genes
type MeasureRelationship struct {
	Type   string   `xml:"Type,attr"`
	Symbol []Symbol `xml:"Symbol"`
}

// Symbol contains gene information
type Symbol struct {
	ElementValue ElementValue `xml:"ElementValue"`
}

// SequenceLocation contains chromosome and position
type SequenceLocation struct {
	Assembly        string `xml:"Assembly,attr"`
	Chr             string `xml:"Chr,attr"`
	Start           int64  `xml:"start,attr"`
	Stop            int64  `xml:"stop,attr"`
	ReferenceAllele string `xml:"referenceAllele,attr"`
	AlternateAllele string `xml:"alternateAllele,attr"`
}

// XRef contains external references (like rsID)
type XRef struct {
	Type string `xml:"Type,attr"`
	ID   string `xml:"ID,attr"`
	DB   string `xml:"DB,attr"`
}

// TraitSet contains associated conditions
type TraitSet struct {
	Type  string  `xml:"Type,attr"`
	Trait []Trait `xml:"Trait"`
}

// Trait contains phenotype/condition information
type Trait struct {
	Type string `xml:"Type,attr"`
	Name []Name `xml:"Name"`
	XRef []XRef `xml:"XRef"`
}

// ObservedIn contains observation metadata
type ObservedIn struct {
	Sample       Sample         `xml:"Sample"`
	Method       []Method       `xml:"Method"`
	ObservedData []ObservedData `xml:"ObservedData"`
}

// Sample contains population information
type Sample struct {
	Origin         string `xml:"Origin"`
	Species        string `xml:"Species"`
	AffectedStatus string `xml:"AffectedStatus"`
}

// Method contains the study method
type Method struct {
	MethodType string `xml:"MethodType"`
}

// ObservedData contains citations
type ObservedData struct {
	Citation []Citation `xml:"Citation"`
}

// Citation contains reference information
type Citation struct {
	Type string `xml:"Type,attr"`
	ID   []ID   `xml:"ID"`
}

// ID contains citation IDs (PubMed, etc.)
type ID struct {
	Source string `xml:"Source,attr"`
	Value  string `xml:",chardata"`
}
