package clinvar

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mkoziy/genome/exporter/internal/models"
)

// mockLimiter is a no-op limiter for tests.
type mockLimiter struct{}

func (mockLimiter) Wait(_ context.Context) error { return nil }
func (mockLimiter) Allow() bool                  { return true }
func (mockLimiter) Reserve() time.Duration       { return 0 }
func (mockLimiter) RetryAfter(int) time.Duration { return 0 }
func (mockLimiter) Reset()                       {}

func TestJoinIDs(t *testing.T) {
	ids := []string{"1", "2", "3"}
	expected := "1,2,3"
	if got := joinIDs(ids); got != expected {
		t.Fatalf("expected %s, got %s", expected, got)
	}
}

func TestQueryBuilder(t *testing.T) {
	q := NewQueryBuilder().WithClinicalSignificance("pathogenic", "likely pathogenic").WithReviewStatus("practice guideline").Build()
	if q == "" {
		t.Fatalf("expected non-empty query")
	}
	if want := "pathogenic[CLNSIG]"; !contains(q, want) {
		t.Fatalf("expected query to contain %s, got %s", want, q)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) > 0 && (contains(s[1:], substr) || s[:len(substr)] == substr))
}

func TestMapToSNPAndClinical(t *testing.T) {
	xmlData := `
	<ClinVarSet>
	  <ReferenceClinVarAssertion>
	    <ClinVarAccession Acc="VCV000000001" Version="1" Type="Variation" />
	    <ClinicalSignificance DateLastEvaluated="2020-01-01">
	      <ReviewStatus>reviewed by expert panel</ReviewStatus>
	      <Description>Pathogenic</Description>
	      <Citation Type="pubmed">
	        <ID Source="PubMed">12345</ID>
	      </Citation>
	    </ClinicalSignificance>
	    <MeasureSet Type="Variant">
	      <Measure Type="SNV">
	        <Name>
	          <ElementValue Type="Preferred">Test Variant</ElementValue>
	        </Name>
	        <AttributeSet>
	          <Attribute Type="MolecularConsequence">missense</Attribute>
	        </AttributeSet>
	        <MeasureRelationship Type="genes overlapped by variant">
	          <Symbol>
	            <ElementValue Type="Preferred">APOE</ElementValue>
	          </Symbol>
	        </MeasureRelationship>
	        <SequenceLocation Assembly="GRCh38" Chr="19" start="44908684" stop="44908685" referenceAllele="C" alternateAllele="T" />
	        <XRef Type="rs" DB="dbSNP" ID="rs429358" />
	      </Measure>
	    </MeasureSet>
	    <TraitSet Type="Phenotype">
	      <Trait Type="Disease">
	        <Name>
	          <ElementValue Type="Preferred">Alzheimer disease</ElementValue>
	        </Name>
	        <XRef Type="MedGen" ID="C0002395" DB="MedGen" />
	      </Trait>
	    </TraitSet>
	    <ObservedIn>
	      <Sample>
	        <Origin>germline</Origin>
	        <Species>human</Species>
	        <AffectedStatus>affected</AffectedStatus>
	      </Sample>
	      <ObservedData>
	        <Citation Type="pubmed">
	          <ID Source="PubMed">67890</ID>
	        </Citation>
	      </ObservedData>
	    </ObservedIn>
	  </ReferenceClinVarAssertion>
	</ClinVarSet>`

	var cvSet ClinVarSet
	if err := xml.Unmarshal([]byte(xmlData), &cvSet); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	snp, err := MapToSNP(cvSet)
	if err != nil {
		t.Fatalf("MapToSNP error: %v", err)
	}
	if snp.RsID != "rs429358" {
		t.Fatalf("unexpected rsid: %s", snp.RsID)
	}
	if snp.Chromosome != "19" || snp.Position != 44908684 {
		t.Fatalf("unexpected position: chr%s:%d", snp.Chromosome, snp.Position)
	}
	if snp.FunctionalClass == nil || *snp.FunctionalClass != models.FuncMissense {
		t.Fatalf("expected functional class missense, got %v", snp.FunctionalClass)
	}

	clin := MapToClinical(cvSet, 1)
	if len(clin) != 1 {
		t.Fatalf("expected 1 clinical record, got %d", len(clin))
	}
	if clin[0].ClinicalSignificance != models.ClinicalPathogenic {
		t.Fatalf("unexpected clinical significance: %s", clin[0].ClinicalSignificance)
	}
	if clin[0].ReviewStatus != models.ReviewExpertPanel {
		t.Fatalf("unexpected review status: %s", clin[0].ReviewStatus)
	}
	if clin[0].ConditionName != "Alzheimer disease" {
		t.Fatalf("unexpected condition: %s", clin[0].ConditionName)
	}
	if clin[0].ConditionID == nil || *clin[0].ConditionID != "C0002395" {
		t.Fatalf("unexpected condition id: %v", clin[0].ConditionID)
	}

	refs := MapToReferences(cvSet, 1)
	if len(refs) != 2 {
		t.Fatalf("expected 2 references, got %d", len(refs))
	}
}

func TestClientSearchAndFetch(t *testing.T) {
	// simple mock server that responds to esearch and efetch
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/esearch.fcgi":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"esearchresult":{"count":"1","retmax":"1","retstart":"0","idlist":["123"],"webenv":"","querykey":""}}`))
		case "/efetch.fcgi":
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<ClinVarResult-Set><ClinVarSet><ReferenceClinVarAssertion><ClinVarAccession Acc="VCV000000001" Version="1" Type="Variation" /><ClinicalSignificance DateLastEvaluated="2020-01-01"><ReviewStatus>reviewed by expert panel</ReviewStatus><Description>Pathogenic</Description></ClinicalSignificance><MeasureSet Type="Variant"><Measure Type="SNV"><SequenceLocation Assembly="GRCh38" Chr="19" start="44908684" stop="44908685" referenceAllele="C" alternateAllele="T" /><XRef Type="rs" DB="dbSNP" ID="rs429358" /></Measure></MeasureSet><TraitSet Type="Phenotype"><Trait Type="Disease"><Name><ElementValue Type="Preferred">Alzheimer disease</ElementValue></Name></Trait></TraitSet></ReferenceClinVarAssertion></ClinVarSet></ClinVarResult-Set>`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	origBase := baseURL
	baseURL = ts.URL
	t.Cleanup(func() { baseURL = origBase })
	client := &Client{httpClient: ts.Client(), limiter: mockLimiter{}}

	search, err := client.Search(context.Background(), "test", 0, 1)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if search.Count != "1" || len(search.IdList) != 1 || search.IdList[0] != "123" {
		t.Fatalf("unexpected search response: %+v", search)
	}

	sets, err := client.Fetch(context.Background(), []string{"123"})
	if err != nil {
		t.Fatalf("fetch error: %v", err)
	}
	if len(sets) != 1 {
		t.Fatalf("expected 1 ClinVarSet, got %d", len(sets))
	}
}

func TestFetcherDeduplicates(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/esearch.fcgi":
			calls++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"esearchresult":{"count":"2","retmax":"2","retstart":"0","idlist":["1","1"],"webenv":"","querykey":""}}`))
		case "/efetch.fcgi":
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<ClinVarResult-Set><ClinVarSet><ReferenceClinVarAssertion><ClinVarAccession Acc="VCV000000001" Version="1" Type="Variation" /><ClinicalSignificance DateLastEvaluated="2020-01-01"><ReviewStatus>reviewed by expert panel</ReviewStatus><Description>Pathogenic</Description></ClinicalSignificance><MeasureSet Type="Variant"><Measure Type="SNV"><SequenceLocation Assembly="GRCh38" Chr="19" start="44908684" stop="44908685" referenceAllele="C" alternateAllele="T" /><XRef Type="rs" DB="dbSNP" ID="rs429358" /></Measure></MeasureSet><TraitSet Type="Phenotype"><Trait Type="Disease"><Name><ElementValue Type="Preferred">Alzheimer disease</ElementValue></Name></Trait></TraitSet></ReferenceClinVarAssertion></ClinVarSet></ClinVarResult-Set>`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	origBase := baseURL
	baseURL = ts.URL
	t.Cleanup(func() { baseURL = origBase })
	client := &Client{httpClient: ts.Client(), limiter: mockLimiter{}}
	fetcher := NewFetcher(client)

	fetcherBase := baseURL
	baseURL = ts.URL
	defer func() { baseURL = fetcherBase }()

	data, err := fetcher.fetchByQuery(context.Background(), "test", make(map[string]bool))
	if err != nil {
		t.Fatalf("fetcher error: %v", err)
	}
	if len(data) != 1 {
		t.Fatalf("expected 1 unique SNP, got %d", len(data))
	}
	if calls != 2 { // initial count search + one batch search
		t.Fatalf("expected esearch called twice, got %d", calls)
	}
}
