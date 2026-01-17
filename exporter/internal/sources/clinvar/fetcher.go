package clinvar

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/mkoziy/genome/exporter/internal/models"
)

// Fetcher orchestrates ClinVar data fetching.
type Fetcher struct {
	client *Client
}

// NewFetcher creates a new ClinVar fetcher.
func NewFetcher(client *Client) *Fetcher {
	return &Fetcher{client: client}
}

// FetchSignificantSNPs fetches all significant SNPs from ClinVar.
func (f *Fetcher) FetchSignificantSNPs(ctx context.Context) ([]SNPData, error) {
	queries := []string{QueryPathogenicVariants(), QueryRiskFactorVariants(), QueryDrugResponseVariants()}

	allData := make([]SNPData, 0)
	seen := make(map[string]bool)

	for _, query := range queries {
		log.Printf("Fetching ClinVar variants for query: %s", query)

		data, err := f.fetchByQuery(ctx, query, seen)
		if err != nil {
			return nil, fmt.Errorf("fetch query: %w", err)
		}

		allData = append(allData, data...)
		log.Printf("Fetched %d unique variants so far", len(allData))
	}

	return allData, nil
}

func (f *Fetcher) fetchByQuery(ctx context.Context, query string, seen map[string]bool) ([]SNPData, error) {
	const batchSize = 500

	searchResp, err := f.client.Search(ctx, query, 0, 1)
	if err != nil {
		return nil, fmt.Errorf("initial search: %w", err)
	}

	totalCount, _ := strconv.Atoi(searchResp.Count)
	log.Printf("Found %d variants", totalCount)

	result := make([]SNPData, 0)

	for start := 0; start < totalCount; start += batchSize {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		searchResp, err := f.client.Search(ctx, query, start, batchSize)
		if err != nil {
			log.Printf("Error searching batch at %d: %v", start, err)
			continue
		}
		if len(searchResp.IdList) == 0 {
			break
		}

		cvSets, err := f.client.Fetch(ctx, searchResp.IdList)
		if err != nil {
			log.Printf("Error fetching batch: %v", err)
			continue
		}

		for _, cvSet := range cvSets {
			snp, err := MapToSNP(cvSet)
			if err != nil {
				log.Printf("Error mapping SNP: %v", err)
				continue
			}
			if seen[snp.RsID] {
				continue
			}
			seen[snp.RsID] = true

			clinical := MapToClinical(cvSet, 0)
			references := MapToReferences(cvSet, 0)

			result = append(result, SNPData{SNP: snp, Clinical: clinical, References: references})
		}

		log.Printf("Processed %d/%d variants", start+len(cvSets), totalCount)
	}

	return result, nil
}

// SNPData bundles all related data for a SNP.
type SNPData struct {
	SNP        *models.SNP
	Clinical   []models.ClinicalData
	References []models.Reference
}
