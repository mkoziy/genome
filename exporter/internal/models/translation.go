package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Translation represents translated content for SNP fields.
type Translation struct {
	bun.BaseModel `bun:"table:snp_translations,alias:t"`

	ID             int64     `bun:"id,pk,autoincrement" json:"id"`
	SNPID          int64     `bun:"snp_id,notnull" json:"snp_id"`
	LanguageCode   string    `bun:"language_code,notnull" json:"language_code"`
	FieldName      string    `bun:"field_name,notnull" json:"field_name"`
	TranslatedText string    `bun:"translated_text,notnull" json:"translated_text"`
	Translator     *string   `bun:"translator" json:"translator,omitempty"`
	TranslatedAt   time.Time `bun:"translated_at,nullzero,notnull,default:current_timestamp" json:"translated_at"`
	Verified       bool      `bun:"verified,default:false" json:"verified"`

	SNP *SNP `bun:"rel:belongs-to,join:snp_id=id" json:"-"`
}

// PhenotypeTranslation represents translated phenotype names.
type PhenotypeTranslation struct {
	bun.BaseModel `bun:"table:phenotype_translations,alias:pt"`

	ID             int64     `bun:"id,pk,autoincrement" json:"id"`
	PhenotypeID    int64     `bun:"phenotype_id,notnull" json:"phenotype_id"`
	LanguageCode   string    `bun:"language_code,notnull" json:"language_code"`
	TranslatedName string    `bun:"translated_name,notnull" json:"translated_name"`
	Translator     *string   `bun:"translator" json:"translator,omitempty"`
	TranslatedAt   time.Time `bun:"translated_at,nullzero,notnull,default:current_timestamp" json:"translated_at"`
	Verified       bool      `bun:"verified,default:false" json:"verified"`

	Phenotype *Phenotype `bun:"rel:belongs-to,join:phenotype_id=id" json:"-"`
}
