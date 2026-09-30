package provider

import (
	"context"
	"encoding/json"
	"errors"
	"inforadar/internal/constants"
	serviceinterface "inforadar/internal/service/interface"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSearchAdaptersAndErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		switch r.URL.Path {
		case "/zhipu":
			if body["search_query"] != "AI 产品" || body["search_recency_filter"] != "oneWeek" {
				t.Errorf("Zhipu request = %+v", body)
			}
			_, _ = w.Write([]byte(`{"search_result":[{"title":"产品更新","content":"摘要","link":"https://example.com/news","publish_date":"2026-09-29"}]}`))
		case "/bailian":
			parameters := body["parameters"].(map[string]any)
			search := parameters["search_options"].(map[string]any)
			if parameters["enable_search"] != true || search["enable_source"] != true || search["forced_search"] != true {
				t.Errorf("Bailian request = %+v", body)
			}
			_, _ = w.Write([]byte(`{"output":{"search_info":{"search_results":[{"title":"发布公告","url":"https://example.com/release"}]}}}`))
		case "/limited":
			w.WriteHeader(http.StatusTooManyRequests)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	c := NewWithEndpoints(server.Client(), Endpoints{ZhipuSearch: server.URL + "/zhipu", BailianSearch: server.URL + "/bailian"})
	zhipu, err := c.Search(context.Background(), constants.ProviderZhipu, "test-key", "AI 产品")
	if err != nil || len(zhipu) != 1 || zhipu[0].Snippet != "摘要" || zhipu[0].PublishedAt == nil || zhipu[0].PublishedAt.Format("2006-01-02") != "2026-09-29" {
		t.Fatalf("Zhipu result = %+v, %v", zhipu, err)
	}
	bailian, err := c.Search(context.Background(), constants.ProviderBailian, "test-key", "AI 产品")
	if err != nil || len(bailian) != 1 || bailian[0].Snippet != "" || bailian[0].PublishedAt != nil {
		t.Fatalf("Bailian result = %+v, %v", bailian, err)
	}
	_, err = c.Search(context.Background(), constants.ProviderZhipu, "wrong-key", "AI 产品")
	if !errors.Is(err, serviceinterface.ErrProviderUnauthorized) {
		t.Fatalf("invalid key = %v", err)
	}
	c.endpoints.ZhipuSearch = server.URL + "/limited"
	_, err = c.Search(context.Background(), constants.ProviderZhipu, "test-key", "AI 产品")
	if !errors.Is(err, serviceinterface.ErrProviderLimited) {
		t.Fatalf("rate limited = %v", err)
	}
}

func TestModelCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model       string              `json:"model"`
			Messages    []map[string]string `json:"messages"`
			Temperature float64             `json:"temperature"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-key" || body.Temperature != 0.1 || len(body.Messages) != 2 ||
			body.Messages[0]["role"] != "system" || body.Messages[1]["role"] != "user" || body.Messages[1]["content"] != "generate" {
			t.Errorf("model request = %+v", body)
		}
		switch r.URL.Path {
		case "/deepseek/chat/completions":
			if body.Model != "deepseek-flash" {
				t.Errorf("DeepSeek model = %q", body.Model)
			}
		case "/zhipu/chat/completions":
			if body.Model != "glm-5.3-flash" {
				t.Errorf("Zhipu model = %q", body.Model)
			}
		case "/bailian/chat/completions":
			if body.Model != "qwen-plus" {
				t.Errorf("Bailian model = %q", body.Model)
			}
		default:
			t.Errorf("unexpected model endpoint %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"query\":\"AI 产品\"}"}}]}`))
	}))
	defer server.Close()
	c := NewWithEndpoints(&http.Client{Timeout: time.Second}, Endpoints{
		DeepSeek:     server.URL + "/deepseek/chat/completions",
		ZhipuModel:   server.URL + "/zhipu/chat/completions",
		BailianModel: server.URL + "/bailian/chat/completions",
	})
	for _, test := range []struct{ provider, model string }{
		{constants.ProviderDeepSeek, "deepseek-flash"},
		{constants.ProviderZhipu, "glm-5.3-flash"},
		{constants.ProviderBailian, "qwen-plus"},
	} {
		t.Run(test.provider, func(t *testing.T) {
			response, err := c.Complete(context.Background(), test.provider, test.model, "test-key", "generate")
			if err != nil || response != `{"query":"AI 产品"}` {
				t.Fatalf("response = %q, %v", response, err)
			}
		})
	}
}

func TestModelCompletionErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status, err := strconv.Atoi(strings.TrimPrefix(strings.TrimSuffix(r.URL.Path, "/chat/completions"), "/status/"))
		if err != nil {
			t.Errorf("unexpected model path %s", r.URL.Path)
			return
		}
		w.WriteHeader(status)
		if status == http.StatusOK {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":" "}}]}`))
		} else {
			_, _ = w.Write([]byte(`{"error":{"message":"provider rejected request"}}`))
		}
	}))
	defer server.Close()
	for _, test := range []struct {
		status int
		want   error
	}{
		{http.StatusBadRequest, serviceinterface.ErrProviderInvalidRequest},
		{http.StatusUnauthorized, serviceinterface.ErrProviderUnauthorized},
		{http.StatusForbidden, serviceinterface.ErrProviderUnauthorized},
		{http.StatusNotFound, serviceinterface.ErrProviderInvalidRequest},
		{http.StatusTooManyRequests, serviceinterface.ErrProviderLimited},
		{http.StatusServiceUnavailable, serviceinterface.ErrProviderUnavailable},
		{http.StatusOK, serviceinterface.ErrProviderUnavailable},
	} {
		t.Run(strconv.Itoa(test.status), func(t *testing.T) {
			c := NewWithEndpoints(server.Client(), Endpoints{DeepSeek: server.URL + "/status/" + strconv.Itoa(test.status) + "/chat/completions"})
			_, err := c.Complete(context.Background(), constants.ProviderDeepSeek, "deepseek-flash", "test-key", "generate")
			if !errors.Is(err, test.want) {
				t.Fatalf("status %d error = %v, want %v", test.status, err, test.want)
			}
		})
	}
	c := NewWithEndpoints(server.Client(), Endpoints{})
	if _, err := c.Complete(context.Background(), constants.ProviderDeepSeek, "deepseek-flash", "test-key", "generate"); !errors.Is(err, serviceinterface.ErrProviderUnavailable) {
		t.Fatalf("missing endpoint error = %v", err)
	}
}
