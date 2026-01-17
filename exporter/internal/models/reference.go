package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Reference represents a research study related to a SNP.
type Reference struct {
	bun.BaseModel `bun:"table:snp_references,alias:r"`

	ID              int64     `bun:"id,pk,autoincrement" json:"id"`
	SNPID           int64     `bun:"snp_id,notnull" json:"snp_id"`
	PubmedID        *string   `bun:"pubmed_id" json:"pubmed_id,omitempty"`
	Title           *string   `bun:"title" json:"title,omitempty"`
	Authors         *string   `bun:"authors" json:"authors,omitempty"`
	Journal         *string   `bun:"journal" json:"journal,omitempty"`
	PublicationYear *int      `bun:"publication_year" json:"publication_year,omitempty"`
	DOI             *string   `bun:"doi" json:"doi,omitempty"`
	URL             *string   `bun:"url" json:"url,omitempty"`
	CitationCount   int       `bun:"citation_count,default:0" json:"citation_count"`
	Abstract        *string   `bun:"abstract" json:"abstract,omitempty"`
	CreatedAt       time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`

	SNP *SNP `bun:"rel:belongs-to,join:snp_id=id" json:"-"`
}

// GetPubmedURL returns the full PubMed URL.
func (r *Reference) GetPubmedURL() string {
	if r.PubmedID == nil {
		return ""
	}
	return "https://pubmed.ncbi.nlm.nih.gov/" + *r.PubmedID
}

// IsHighlyCited returns true if citation count > 100.
func (r *Reference) IsHighlyCited() bool {
	return r.CitationCount > 100
}
