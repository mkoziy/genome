# 004 - ClinVar Data Source

## Feature Overview
Implement data fetcher for ClinVar, NCBI's public archive of reports on relationships among medically important variants and phenotypes. ClinVar provides high-quality clinical significance data that will be the primary source for calculating SNP significance scores.

## Goals
- Fetch SNPs with clinical significance from ClinVar
- Extract all relevant clinical annotations
- Parse variant details (position, alleles, genes)
- Collect associated conditions and phenotypes
- Store PubMed references
- Handle API pagination and errors
- Respect NCBI rate limits

## ClinVar API Details

### Base Information
- **API**: NCBI E-utilities (Entrez)
- **Endpoint**: https://eutils.ncbi.nlm.nih.gov/entrez/eutils/
- **Databases**: clinvar, SNP
- **Rate Limit**: 3 req/sec (no key), 10 req/sec (with API key)
- **Response Format**: XML, JSON (limited)
- **Documentation**: https://www.ncbi.nlm.nih.gov/clinvar/docs/help/

### E-utilities Components

1. **ESearch**: Search ClinVar database
2. **ESummary**: Get summary of variants
3. **EFetch**: Get full variant details in XML

### Query Strategy

#### Step 1: Search for Pathogenic/Likely Pathogenic Variants
```
https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esearch.fcgi
?db=clinvar
&term=(pathogenic[CLNSIG] OR likely pathogenic[CLNSIG])
&retmax=10000
&retmode=json
```

#### Step 2: Fetch Details for Each Variant
```
https://eutils.ncbi.nlm.nih.gov/entrez/eutils/efetch.fcgi
?db=clinvar
&id=12345,67890
&rettype=vcv
&retmode=xml
```

## Package Structure

```
internal/sources/clinvar/
├── client.go           # HTTP client with rate limiting
├── parser.go           # XML parsing logic
├── mapper.go           # Map ClinVar data to models
├── fetcher.go          # Main fetcher orchestration
├── queries.go          # Query builders
├── types.go            # ClinVar-specific types
└── clinvar_test.go     # Tests
```

## Implementation

### types.go - ClinVar-Specific Types

```go
package clinvar

import (
    "encoding/xml"
)

// SearchResponse represents ESearch response
type SearchResponse struct {
    Count      string   `json:"count"`
    RetMax     string   `json:"retmax"`
    RetStart   string   `json:"retstart"`
    IdList     []string `json:"idlist"`
    WebEnv     string   `json:"webenv"`
    QueryKey   string   `json:"querykey"`
}

// ClinVarSet is the root element for variant details
type ClinVarSet struct {
    XMLName         xml.Name        `xml:"ClinVarSet"`
    ReferenceClinVarAssertion ReferenceClinVarAssertion `xml:"ReferenceClinVarAssertion"`
}

// ReferenceClinVarAssertion contains the main variant information
type ReferenceClinVarAssertion struct {
    ClinVarAccession ClinVarAccession `xml:"ClinVarAccession"`
    ClinicalSignificance ClinicalSignificance `xml:"ClinicalSignificance"`
    MeasureSet      MeasureSet       `xml:"MeasureSet"`
    TraitSet        TraitSet         `xml:"TraitSet"`
    ObservedIn      []ObservedIn     `xml:"ObservedIn"`
}

// ClinVarAccession contains the variant ID
type ClinVarAccession struct {
    Acc     string `xml:"Acc,attr"`
    Version int    `xml:"Version,attr"`
    Type    string `xml:"Type,attr"`
}

// ClinicalSignificance contains the clinical interpretation
type ClinicalSignificance struct {
    ReviewStatus    string          `xml:"ReviewStatus"`
    Description     string          `xml:"Description"`
    DateLastEvaluated string        `xml:"DateLastEvaluated,attr"`
    Citation        []Citation      `xml:"Citation"`
}

// MeasureSet contains one or more variants
type MeasureSet struct {
    Type    string    `xml:"Type,attr"`
    Measure []Measure `xml:"Measure"`
}

// Measure contains variant details
type Measure struct {
    Type             string           `xml:"Type,attr"`
    Name             []Name           `xml:"Name"`
    AttributeSet     []AttributeSet   `xml:"AttributeSet"`
    MeasureRelationship []MeasureRelationship `xml:"MeasureRelationship"`
    SequenceLocation []SequenceLocation `xml:"SequenceLocation"`
    XRef             []XRef           `xml:"XRef"`
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
    Type   string  `xml:"Type,attr"`
    Symbol []Symbol `xml:"Symbol"`
}

// Symbol contains gene information
type Symbol struct {
    ElementValue ElementValue `xml:"ElementValue"`
}

// SequenceLocation contains chromosome and position
type SequenceLocation struct {
    Assembly         string `xml:"Assembly,attr"`
    Chr              string `xml:"Chr,attr"`
    Start            int64  `xml:"start,attr"`
    Stop             int64  `xml:"stop,attr"`
    ReferenceAllele  string `xml:"referenceAllele,attr"`
    AlternateAllele  string `xml:"alternateAllele,attr"`
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
    Type string  `xml:"Type,attr"`
    Name []Name  `xml:"Name"`
    XRef []XRef  `xml:"XRef"`
}

// ObservedIn contains observation metadata
type ObservedIn struct {
    Sample        Sample        `xml:"Sample"`
    Method        []Method      `xml:"Method"`
    ObservedData  []ObservedData `xml:"ObservedData"`
}

// Sample contains population information
type Sample struct {
    Origin       string `xml:"Origin"`
    Species      string `xml:"Species"`
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
    Type    string  `xml:"Type,attr"`
    ID      []ID    `xml:"ID"`
}

// ID contains citation IDs (PubMed, etc.)
type ID struct {
    Source string `xml:"Source,attr"`
    Value  string `xml:",chardata"`
}
```

### client.go - HTTP Client

```go
package clinvar

import (
    "context"
    "encoding/json"
    "encoding/xml"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "time"
    
    "snp-downloader/internal/ratelimit"
)

const (
    BaseURL = "https://eutils.ncbi.nlm.nih.gov/entrez/eutils"
    ToolName = "snp-downloader"
    ToolEmail = "" // Set via config
)

// Client handles ClinVar API requests
type Client struct {
    httpClient *http.Client
    limiter    ratelimit.Limiter
    apiKey     string
    email      string
}

// NewClient creates a new ClinVar client
func NewClient(limiter ratelimit.Limiter, apiKey, email string) *Client {
    return &Client{
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
        limiter: limiter,
        apiKey:  apiKey,
        email:   email,
    }
}

// Search performs an ESearch query
func (c *Client) Search(ctx context.Context, query string, retStart, retMax int) (*SearchResponse, error) {
    // Wait for rate limiter
    if err := c.limiter.Wait(ctx); err != nil {
        return nil, err
    }
    
    // Build URL
    params := url.Values{}
    params.Set("db", "clinvar")
    params.Set("term", query)
    params.Set("retstart", fmt.Sprintf("%d", retStart))
    params.Set("retmax", fmt.Sprintf("%d", retMax))
    params.Set("retmode", "json")
    params.Set("tool", ToolName)
    
    if c.email != "" {
        params.Set("email", c.email)
    }
    if c.apiKey != "" {
        params.Set("api_key", c.apiKey)
    }
    
    url := fmt.Sprintf("%s/esearch.fcgi?%s", BaseURL, params.Encode())
    
    // Make request
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("execute request: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }
    
    // Parse response
    var result struct {
        ESearchResult SearchResponse `json:"esearchresult"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }
    
    return &result.ESearchResult, nil
}

// Fetch retrieves full variant details by IDs
func (c *Client) Fetch(ctx context.Context, ids []string) ([]ClinVarSet, error) {
    if len(ids) == 0 {
        return nil, nil
    }
    
    // Wait for rate limiter
    if err := c.limiter.Wait(ctx); err != nil {
        return nil, err
    }
    
    // Build URL
    params := url.Values{}
    params.Set("db", "clinvar")
    params.Set("id", joinIDs(ids))
    params.Set("rettype", "vcv")
    params.Set("retmode", "xml")
    params.Set("tool", ToolName)
    
    if c.email != "" {
        params.Set("email", c.email)
    }
    if c.apiKey != "" {
        params.Set("api_key", c.apiKey)
    }
    
    url := fmt.Sprintf("%s/efetch.fcgi?%s", BaseURL, params.Encode())
    
    // Make request
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("execute request: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
    }
    
    // Parse XML
    var wrapper struct {
        XMLName xml.Name     `xml:"ClinVarResult-Set"`
        Sets    []ClinVarSet `xml:"ClinVarSet"`
    }
    
    if err := xml.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
        return nil, fmt.Errorf("decode XML: %w", err)
    }
    
    return wrapper.Sets, nil
}

func joinIDs(ids []string) string {
    result := ""
    for i, id := range ids {
        if i > 0 {
            result += ","
        }
        result += id
    }
    return result
}
```

### queries.go - Query Builders

```go
package clinvar

import (
    "fmt"
    "strings"
)

// QueryBuilder builds ClinVar search queries
type QueryBuilder struct {
    terms []string
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder() *QueryBuilder {
    return &QueryBuilder{
        terms: make([]string, 0),
    }
}

// WithClinicalSignificance adds clinical significance filter
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

// WithReviewStatus adds review status filter
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

// WithVariantType adds variant type filter
func (qb *QueryBuilder) WithVariantType(varType string) *QueryBuilder {
    if varType != "" {
        qb.terms = append(qb.terms, fmt.Sprintf("%s[VARTYPE]", varType))
    }
    return qb
}

// WithGene adds gene symbol filter
func (qb *QueryBuilder) WithGene(gene string) *QueryBuilder {
    if gene != "" {
        qb.terms = append(qb.terms, fmt.Sprintf("%s[GENE]", gene))
    }
    return qb
}

// Build constructs the final query string
func (qb *QueryBuilder) Build() string {
    if len(qb.terms) == 0 {
        return ""
    }
    return strings.Join(qb.terms, " AND ")
}

// Predefined queries
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
```

### mapper.go - Data Mapping

```go
package clinvar

import (
    "fmt"
    "strconv"
    "strings"
    "time"
    
    "snp-downloader/internal/models"
)

// MapToSNP converts ClinVarSet to SNP model
func MapToSNP(cvSet ClinVarSet) (*models.SNP, error) {
    ref := cvSet.ReferenceClinVarAssertion
    
    if len(ref.MeasureSet.Measure) == 0 {
        return nil, fmt.Errorf("no measures in variant")
    }
    
    measure := ref.MeasureSet.Measure[0]
    
    // Extract rsID
    rsID := extractRsID(measure.XRef)
    if rsID == "" {
        return nil, fmt.Errorf("no rsID found")
    }
    
    // Extract position and alleles
    seqLoc := findGRCh38Location(measure.SequenceLocation)
    if seqLoc == nil {
        return nil, fmt.Errorf("no GRCh38 location found")
    }
    
    // Extract gene
    geneSymbol := extractGeneSymbol(measure.MeasureRelationship)
    
    // Determine variant type
    varType := models.SNV
    if measure.Type != "" {
        varType = models.VariantType(measure.Type)
    }
    
    // Determine functional class
    funcClass := extractFunctionalClass(measure.AttributeSet)
    
    snp := &models.SNP{
        RsID:            rsID,
        Chromosome:      seqLoc.Chr,
        Position:        seqLoc.Start,
        ReferenceAllele: seqLoc.ReferenceAllele,
        AlternateAlleles: []string{seqLoc.AlternateAllele},
        GeneSymbol:      geneSymbol,
        VariantType:     varType,
        FunctionalClass: funcClass,
    }
    
    return snp, nil
}

// MapToClinical converts clinical significance data
func MapToClinical(cvSet ClinVarSet, snpID int64) []models.ClinicalData {
    ref := cvSet.ReferenceClinVarAssertion
    clinSig := ref.ClinicalSignificance
    
    result := make([]models.ClinicalData, 0)
    
    // Create entry for each trait/condition
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

// MapToReferences extracts PubMed references
func MapToReferences(cvSet ClinVarSet, snpID int64) []models.Reference {
    refs := make([]models.Reference, 0)
    seen := make(map[string]bool)
    
    // Extract from clinical significance citations
    for _, citation := range cvSet.ReferenceClinVarAssertion.ClinicalSignificance.Citation {
        pmid := extractPubMedID(citation.ID)
        if pmid != "" && !seen[pmid] {
            refs = append(refs, models.Reference{
                SNPID:    snpID,
                PubmedID: &pmid,
            })
            seen[pmid] = true
        }
    }
    
    // Extract from observed data
    for _, obs := range cvSet.ReferenceClinVarAssertion.ObservedIn {
        for _, obsData := range obs.ObservedData {
            for _, citation := range obsData.Citation {
                pmid := extractPubMedID(citation.ID)
                if pmid != "" && !seen[pmid] {
                    refs = append(refs, models.Reference{
                        SNPID:    snpID,
                        PubmedID: &pmid,
                    })
                    seen[pmid] = true
                }
            }
        }
    }
    
    return refs
}

// Helper functions

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
        return models.LikelyPathogenic
    case strings.Contains(desc, "pathogenic"):
        return models.Pathogenic
    case strings.Contains(desc, "benign") && strings.Contains(desc, "likely"):
        return models.LikelyBenign
    case strings.Contains(desc, "benign"):
        return models.Benign
    case strings.Contains(desc, "risk"):
        return models.RiskFactor
    case strings.Contains(desc, "protective"):
        return models.Protective
    case strings.Contains(desc, "drug"):
        return models.DrugResponse
    case strings.Contains(desc, "uncertain"):
        return models.UncertainSignificance
    default:
        return models.Other
    }
}

func mapReviewStatus(status string) models.ReviewStatus {
    status = strings.ToLower(strings.TrimSpace(status))
    switch {
    case strings.Contains(status, "practice guideline"):
        return models.PracticeGuideline
    case strings.Contains(status, "expert panel"):
        return models.ReviewedByExpertPanel
    case strings.Contains(status, "criteria provided"):
        return models.CriteriaProvided
    case strings.Contains(status, "multiple"):
        return models.MultipleSubmitters
    case strings.Contains(status, "single"):
        return models.SingleSubmitter
    default:
        return models.NoAssertion
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
```

### fetcher.go - Main Orchestration

```go
package clinvar

import (
    "context"
    "fmt"
    "log"
    
    "snp-downloader/internal/models"
)

// Fetcher orchestrates ClinVar data fetching
type Fetcher struct {
    client *Client
}

// NewFetcher creates a new ClinVar fetcher
func NewFetcher(client *Client) *Fetcher {
    return &Fetcher{
        client: client,
    }
}

// FetchSignificantSNPs fetches all significant SNPs from ClinVar
func (f *Fetcher) FetchSignificantSNPs(ctx context.Context) ([]SNPData, error) {
    queries := []string{
        QueryPathogenicVariants(),
        QueryRiskFactorVariants(),
        QueryDrugResponseVariants(),
    }
    
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
    
    // Search for variant IDs
    searchResp, err := f.client.Search(ctx, query, 0, 1)
    if err != nil {
        return nil, fmt.Errorf("initial search: %w", err)
    }
    
    totalCount, _ := strconv.Atoi(searchResp.Count)
    log.Printf("Found %d variants", totalCount)
    
    result := make([]SNPData, 0)
    
    // Fetch in batches
    for start := 0; start < totalCount; start += batchSize {
        select {
        case <-ctx.Done():
            return result, ctx.Err()
        default:
        }
        
        // Search for batch
        searchResp, err := f.client.Search(ctx, query, start, batchSize)
        if err != nil {
            log.Printf("Error searching batch at %d: %v", start, err)
            continue
        }
        
        if len(searchResp.IdList) == 0 {
            break
        }
        
        // Fetch details
        cvSets, err := f.client.Fetch(ctx, searchResp.IdList)
        if err != nil {
            log.Printf("Error fetching batch: %v", err)
            continue
        }
        
        // Process each variant
        for _, cvSet := range cvSets {
            snp, err := MapToSNP(cvSet)
            if err != nil {
                log.Printf("Error mapping SNP: %v", err)
                continue
            }
            
            // Skip duplicates
            if seen[snp.RsID] {
                continue
            }
            seen[snp.RsID] = true
            
            // Extract related data
            clinical := MapToClinical(cvSet, 0) // snpID will be set later
            references := MapToReferences(cvSet, 0)
            
            result = append(result, SNPData{
                SNP:        snp,
                Clinical:   clinical,
                References: references,
            })
        }
        
        log.Printf("Processed %d/%d variants", start+len(cvSets), totalCount)
    }
    
    return result, nil
}

// SNPData bundles all related data for a SNP
type SNPData struct {
    SNP        *models.SNP
    Clinical   []models.ClinicalData
    References []models.Reference
}
```

## Configuration

```yaml
clinvar:
  enabled: true
  api_key: ""  # Optional, increases rate limit
  email: ""    # Required if using API
  batch_size: 500
  max_results: 10000
  queries:
    - pathogenic
    - risk_factor
    - drug_response
```

## Implementation Tasks

1. Implement XML parsing for ClinVar responses
2. Create HTTP client with rate limiting
3. Build query construction helpers
4. Implement data mapping to models
5. Create main fetcher orchestration
6. Add error handling and retries
7. Write comprehensive tests with mock data
8. Add progress logging

## Testing

- Mock ClinVar API responses
- Test XML parsing edge cases
- Verify data mapping correctness
- Test pagination handling
- Test rate limiting compliance

## Success Criteria
- Successfully fetch 5000+ pathogenic/likely pathogenic SNPs
- All SNPs have rsID, position, and alleles
- Clinical significance correctly mapped
- References extracted
- Respects rate limits
- Handles errors gracefully
- Progress logging clear

## Next Feature
After completing this, proceed to **005 - dbSNP Data Source**.
