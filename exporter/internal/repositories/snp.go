package repositories

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/mkoziy/genome/exporter/internal/models"
)

// GetSNPByRsID fetches a SNP by rsID with related data.
func GetSNPByRsID(ctx context.Context, db *bun.DB, rsID string) (*models.SNP, error) {
	snp := new(models.SNP)
	err := db.NewSelect().
		Model(snp).
		Where("rsid = ?", rsID).
		Relation("Significance").
		Relation("ClinicalData").
		Relation("Phenotypes").
		Relation("References").
		Relation("PopulationData").
		Scan(ctx)

	return snp, err
}

// GetTopSignificantSNPs returns SNPs ordered by total score with pathogenic clinical annotations.
func GetTopSignificantSNPs(ctx context.Context, db *bun.DB, limit int) ([]*models.SNP, error) {
	var snps []*models.SNP
	err := db.NewSelect().
		Model(&snps).
		Relation("Significance").
		Relation("ClinicalData", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("clinical_significance IN (?)", bun.In([]string{"pathogenic", "likely_pathogenic"}))
		}).
		Join("JOIN snp_significance AS sig ON sig.snp_id = s.id").
		Where("sig.total_score >= ?", 60).
		OrderExpr("sig.total_score DESC").
		Limit(limit).
		Scan(ctx)

	return snps, err
}

// InsertSNPWithData inserts an SNP and related clinical data and references in a transaction.
func InsertSNPWithData(ctx context.Context, db *bun.DB, snp *models.SNP, clinical []*models.ClinicalData, refs []*models.Reference) error {
	return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(snp).Exec(ctx); err != nil {
			return err
		}

		for _, c := range clinical {
			c.SNPID = snp.ID
		}
		for _, r := range refs {
			r.SNPID = snp.ID
		}

		if len(clinical) > 0 {
			if _, err := tx.NewInsert().Model(&clinical).Exec(ctx); err != nil {
				return err
			}
		}

		if len(refs) > 0 {
			if _, err := tx.NewInsert().Model(&refs).Exec(ctx); err != nil {
				return err
			}
		}

		return nil
	})
}

// UpsertSNPs performs a batch upsert on SNPs keyed by rsID.
func UpsertSNPs(ctx context.Context, db *bun.DB, snps []*models.SNP) error {
	_, err := db.NewInsert().
		Model(&snps).
		On("CONFLICT (rsid) DO UPDATE").
		Set("chromosome = EXCLUDED.chromosome").
		Set("position = EXCLUDED.position").
		Set("reference_allele = EXCLUDED.reference_allele").
		Set("alternate_alleles = EXCLUDED.alternate_alleles").
		Set("gene_symbol = EXCLUDED.gene_symbol").
		Set("updated_at = CURRENT_TIMESTAMP").
		Exec(ctx)

	return err
}
