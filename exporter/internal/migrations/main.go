package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"

	"github.com/mkoziy/genome/exporter/internal/models"
)

var Migrations = migrate.NewMigrations()

func init() {
	// Migration 1: create tables
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		modelsList := []interface{}{
			(*models.SNP)(nil),
			(*models.Significance)(nil),
			(*models.ClinicalData)(nil),
			(*models.Phenotype)(nil),
			(*models.Reference)(nil),
			(*models.PopulationFreq)(nil),
			(*models.Translation)(nil),
			(*models.PhenotypeTranslation)(nil),
			(*models.SourceMetadata)(nil),
			(*models.DownloadMetadata)(nil),
		}

		for _, model := range modelsList {
			if _, err := db.NewCreateTable().Model(model).IfNotExists().Exec(ctx); err != nil {
				return err
			}
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		modelsList := []interface{}{
			(*models.DownloadMetadata)(nil),
			(*models.SourceMetadata)(nil),
			(*models.PhenotypeTranslation)(nil),
			(*models.Translation)(nil),
			(*models.PopulationFreq)(nil),
			(*models.Reference)(nil),
			(*models.Phenotype)(nil),
			(*models.ClinicalData)(nil),
			(*models.Significance)(nil),
			(*models.SNP)(nil),
		}

		for _, model := range modelsList {
			if _, err := db.NewDropTable().Model(model).IfExists().Exec(ctx); err != nil {
				return err
			}
		}

		return nil
	})

	// Migration 2: indexes
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		indexes := []string{
			"CREATE INDEX IF NOT EXISTS idx_snps_chromosome_position ON snps(chromosome, position)",
			"CREATE INDEX IF NOT EXISTS idx_snps_gene_symbol ON snps(gene_symbol)",
			"CREATE INDEX IF NOT EXISTS idx_significance_score ON snp_significance(total_score DESC)",
			"CREATE INDEX IF NOT EXISTS idx_clinical_significance ON snp_clinical(clinical_significance)",
			"CREATE INDEX IF NOT EXISTS idx_clinical_condition ON snp_clinical(condition_name)",
			"CREATE INDEX IF NOT EXISTS idx_phenotypes_name ON snp_phenotypes(phenotype_name)",
			"CREATE INDEX IF NOT EXISTS idx_references_pubmed ON snp_references(pubmed_id)",
			"CREATE INDEX IF NOT EXISTS idx_populations_code ON snp_populations(population_code)",
			"CREATE INDEX IF NOT EXISTS idx_translations_snp_lang ON snp_translations(snp_id, language_code)",
		}

		for _, idx := range indexes {
			if _, err := db.ExecContext(ctx, idx); err != nil {
				return err
			}
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		indexes := []string{
			"DROP INDEX IF EXISTS idx_snps_chromosome_position",
			"DROP INDEX IF EXISTS idx_snps_gene_symbol",
			"DROP INDEX IF EXISTS idx_significance_score",
			"DROP INDEX IF EXISTS idx_clinical_significance",
			"DROP INDEX IF EXISTS idx_clinical_condition",
			"DROP INDEX IF EXISTS idx_phenotypes_name",
			"DROP INDEX IF EXISTS idx_references_pubmed",
			"DROP INDEX IF EXISTS idx_populations_code",
			"DROP INDEX IF EXISTS idx_translations_snp_lang",
		}

		for _, idx := range indexes {
			if _, err := db.ExecContext(ctx, idx); err != nil {
				return err
			}
		}

		return nil
	})
}

// RunMigrations runs all pending migrations.
func RunMigrations(ctx context.Context, db *bun.DB) error {
	migrator := migrate.NewMigrator(db, Migrations)

	if err := migrator.Init(ctx); err != nil {
		return err
	}

	group, err := migrator.Migrate(ctx)
	if err != nil {
		return err
	}

	if group.IsZero() {
		fmt.Println("No new migrations to run")
		return nil
	}

	fmt.Printf("Migrated to %s\n", group)
	return nil
}
