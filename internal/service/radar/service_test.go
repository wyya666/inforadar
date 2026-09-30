package radar

import (
	"context"
	"strings"
	"testing"
	"time"

	radarmodel "inforadar/internal/api/model/radar"
	"inforadar/internal/constants"
	repositoryinterface "inforadar/internal/repository/interface"
	cryptoutil "inforadar/internal/utils/crypto"
)

type scanRepo struct {
	repositoryinterface.RadarRepository
	settings    radarmodel.Settings
	credentials map[string]radarmodel.Credential
	query       string
	seen        map[string]bool
	rows        []radarmodel.Candidate
}

func (r *scanRepo) SettingsByUser(context.Context, uint64) (radarmodel.Settings, error) {
	return r.settings, nil
}
func (r *scanRepo) CredentialByUser(_ context.Context, _ uint64, name string) (radarmodel.Credential, error) {
	return r.credentials[name], nil
}
func (r *scanRepo) SaveQuery(_ context.Context, _ uint64, query string) error {
	r.query = query
	return nil
}
func (r *scanRepo) ExistingURLHashes(_ context.Context, _ uint64, hashes []string) (map[string]bool, error) {
	found := map[string]bool{}
	for _, hash := range hashes {
		if r.seen[hash] {
			found[hash] = true
		}
	}
	return found, nil
}
func (r *scanRepo) SaveCandidates(_ context.Context, rows []radarmodel.Candidate) error {
	for _, row := range rows {
		r.seen[row.URLHash] = true
		r.rows = append(r.rows, row)
	}
	return nil
}

type fakeModel struct{ calls int }

func (m *fakeModel) Complete(_ context.Context, _, _, _, prompt string) (string, error) {
	m.calls++
	if strings.Contains(prompt, "提取一个适合网页搜索") {
		return `{"query":"AI 搜索"}`, nil
	}
	return `{"items":[{"index":0,"important":true,"summary":"新产品发布"},{"index":1,"important":false,"summary":"无关"}]}`, nil
}

type fakeSearch struct{ calls int }

func (s *fakeSearch) Search(_ context.Context, _, _, _ string) ([]radarmodel.SearchResult, error) {
	s.calls++
	return []radarmodel.SearchResult{
		{URL: "https://EXAMPLE.com/news?utm_source=search", Title: "AI 搜索产品发布", Snippet: "官方发布摘要"},
		{URL: "https://example.com/other", Title: "无关消息", Snippet: "其他行业"},
	}, nil
}

func TestScanStoresImportantSourcesOnce(t *testing.T) {
	box, err := cryptoutil.NewSecretBox(strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, nonce, err := box.Encrypt(3, constants.ProviderZhipu, "test-api-key")
	if err != nil {
		t.Fatal(err)
	}
	repo := &scanRepo{settings: radarmodel.Settings{UserID: 3, ModelProvider: constants.ProviderZhipu, ModelID: constants.DefaultZhipuModel, SearchProvider: constants.ProviderZhipu, IntervalHours: 6}, credentials: map[string]radarmodel.Credential{constants.ProviderZhipu: {UserID: 3, Provider: constants.ProviderZhipu, Ciphertext: ciphertext, Nonce: nonce}}, seen: map[string]bool{}}
	model, search := &fakeModel{}, &fakeSearch{}
	service := New(repo, model, search, box).(*Service)
	topic := radarmodel.Topic{ID: 9, UserID: 3, Description: "关注 AI 搜索产品发布"}
	count, interval, err := service.scan(context.Background(), &topic, time.Now().UTC())
	if err != nil || count != 1 || interval != 6 || repo.query != "AI 搜索" || len(repo.rows) != 2 {
		t.Fatalf("scan: count=%d interval=%d query=%q rows=%d err=%v", count, interval, repo.query, len(repo.rows), err)
	}
	if !repo.rows[0].Important || repo.rows[1].Important || repo.rows[0].URL != "https://example.com/news" {
		t.Fatalf("candidates = %+v", repo.rows)
	}
	topic.SearchQuery = &repo.query
	count, _, err = service.scan(context.Background(), &topic, time.Now().UTC())
	if err != nil || count != 0 || len(repo.rows) != 2 || model.calls != 2 || search.calls != 2 {
		t.Fatalf("duplicate scan: count=%d rows=%d model=%d search=%d err=%v", count, len(repo.rows), model.calls, search.calls, err)
	}
}

func TestCanonicalURLRejectsUnsafeAndStripsTracking(t *testing.T) {
	for _, raw := range []string{"javascript:alert(1)", "file:///etc/passwd", "https://user:pass@example.com/", "https://"} {
		if _, ok := canonicalURL(raw); ok {
			t.Errorf("accepted %q", raw)
		}
	}
	got, ok := canonicalURL("https://EXAMPLE.com:443/a?utm_source=x&item=1#part")
	if !ok || got != "https://example.com/a?item=1" {
		t.Fatalf("canonical URL = %q, %t", got, ok)
	}
}
