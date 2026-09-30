package radar_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	drivermysql "github.com/go-sql-driver/mysql"
	radarmodel "inforadar/internal/api/model/radar"
	"inforadar/internal/common"
	"inforadar/internal/config"
	"inforadar/internal/constants"
	userrepository "inforadar/internal/repository/user"
	radarsvc "inforadar/internal/service/radar"
	cryptoutil "inforadar/internal/utils/crypto"
)

type integrationModel struct{}

func (integrationModel) Complete(_ context.Context, _, _, _, prompt string) (string, error) {
	if strings.Contains(prompt, "提取一个适合网页搜索") {
		return `{"query":"模型发布"}`, nil
	}
	return `{"items":[{"index":0,"important":true,"summary":"新的模型发布"}]}`, nil
}

type integrationSearch struct{}

func (integrationSearch) Search(context.Context, string, string, string) ([]radarmodel.SearchResult, error) {
	return []radarmodel.SearchResult{{URL: "https://example.com/model-release", Title: "模型发布公告", Snippet: "官方公布新模型"}}, nil
}

// Run only against a disposable MySQL 8 database with both manual scripts applied.
func TestMySQLRadarIsolationAndLeaseRecovery(t *testing.T) {
	dbName := os.Getenv("INFORADAR_TEST_DATABASE_DB")
	if dbName == "" {
		t.Skip("set INFORADAR_TEST_DATABASE_* for a disposable MySQL database")
	}
	port := constants.DefaultMySQLPort
	if raw := os.Getenv("INFORADAR_TEST_DATABASE_PORT"); raw != "" {
		var err error
		port, err = strconv.Atoi(raw)
		if err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.Database{Host: os.Getenv("INFORADAR_TEST_DATABASE_HOST"), Port: port, Username: os.Getenv("INFORADAR_TEST_DATABASE_USERNAME"), Password: os.Getenv("INFORADAR_TEST_DATABASE_PASSWORD"), DB: dbName}
	if cfg.Host == "" || cfg.Username == "" {
		t.Fatal("test database host and username are required")
	}
	ctx := context.Background()
	store, err := userrepository.OpenMySQL(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	driverCfg := drivermysql.NewConfig()
	driverCfg.User, driverCfg.Passwd, driverCfg.Net, driverCfg.Addr, driverCfg.DBName = cfg.Username, cfg.Password, "tcp", net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)), cfg.DB
	driverCfg.ParseTime = true
	driverCfg.Loc = time.UTC
	connector, err := drivermysql.NewConnector(driverCfg)
	if err != nil {
		t.Fatal(err)
	}
	db := sql.OpenDB(connector)
	t.Cleanup(func() { _ = db.Close() })
	createUser := func() uint64 {
		email := fmt.Sprintf("radar-test-%d@example.test", time.Now().UnixNano())
		result, err := db.ExecContext(ctx, "INSERT INTO `user` (user_account,email,user_password,user_name,user_role) VALUES (?,?,?,?,?)", email, email, "test-hash", "Test", "user")
		if err != nil {
			t.Fatal(err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _, _ = db.ExecContext(context.Background(), "DELETE FROM `user` WHERE id = ?", id) })
		return uint64(id)
	}
	firstUser, secondUser := createUser(), createUser()
	repo := store.Radar()
	now := time.Now().UTC()
	for _, id := range []uint64{firstUser, secondUser} {
		if err := repo.SaveSettings(ctx, &radarmodel.Settings{UserID: id, ModelProvider: constants.ProviderZhipu, ModelID: constants.DefaultZhipuModel, SearchProvider: constants.ProviderZhipu, IntervalHours: 6, CreatedAt: now, UpdatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	box, err := cryptoutil.NewSecretBox(strings.Repeat("b", 64))
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, nonce, err := box.Encrypt(firstUser, constants.ProviderZhipu, "private-test-api-key")
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveCredential(ctx, &radarmodel.Credential{UserID: firstUser, Provider: constants.ProviderZhipu, Ciphertext: ciphertext, Nonce: nonce, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	var stored []byte
	if err := db.QueryRowContext(ctx, "SELECT ciphertext FROM provider_credentials WHERE user_id = ? AND provider = ?", firstUser, constants.ProviderZhipu).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stored), "private-test-api-key") {
		t.Fatal("plaintext API key stored in database")
	}
	first := radarmodel.Topic{UserID: firstUser, Description: "first user topic", Status: constants.TopicActive, NextRunAt: &now, CreatedAt: now, UpdatedAt: now}
	second := radarmodel.Topic{UserID: secondUser, Description: "second user topic", Status: constants.TopicActive, NextRunAt: &now, CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateTopic(ctx, &first); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateTopic(ctx, &second); err != nil {
		t.Fatal(err)
	}
	claimed1, run1, err := repo.ClaimDue(ctx, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	claimed2, run2, err := repo.ClaimDue(ctx, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if claimed1.ID == claimed2.ID {
		t.Fatal("same topic claimed twice")
	}
	if _, _, err := repo.ClaimDue(ctx, now.Add(time.Second)); !errors.Is(err, common.ErrNotFound) {
		t.Fatalf("third claim = %v", err)
	}
	if err := repo.FinishRun(ctx, claimed1, run1, constants.RunSucceeded, 0, "", nil, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE monitor_topics SET lease_until = ? WHERE id = ?", now.Add(-time.Minute), claimed2.ID); err != nil {
		t.Fatal(err)
	}
	recovered, recoveryRun, err := repo.ClaimDue(ctx, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if recovered.ID != claimed2.ID || recoveryRun.ID == run2.ID {
		t.Fatal("expired lease was not recovered")
	}
	if err := repo.FinishRun(ctx, recovered, recoveryRun, constants.RunSucceeded, 0, "", nil, 0); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveCandidates(ctx, []radarmodel.Candidate{{UserID: firstUser, TopicID: first.ID, URLHash: strings.Repeat("a", 64), URL: "https://example.com/first", Title: "first", Important: true, FirstSeenAt: now}, {UserID: secondUser, TopicID: second.ID, URLHash: strings.Repeat("b", 64), URL: "https://example.com/second", Title: "second", Important: true, FirstSeenAt: now}}); err != nil {
		t.Fatal(err)
	}
	feed, _, err := repo.ListFeed(ctx, firstUser, 0, 0, 20)
	if err != nil || len(feed) != 1 || feed[0].UserID != firstUser {
		t.Fatalf("isolated feed = %+v, %v", feed, err)
	}
	if _, err := repo.TopicByID(ctx, firstUser, second.ID); !errors.Is(err, common.ErrNotFound) {
		t.Fatalf("other user's topic = %v", err)
	}
	if err := repo.QueueManualScan(ctx, firstUser, first.ID, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := repo.QueueManualScan(ctx, firstUser, first.ID, now.Add(2*time.Minute)); !errors.Is(err, common.ErrRadarCooldown) {
		t.Fatalf("manual cooldown = %v", err)
	}
	firstRow, err := repo.TopicByID(ctx, firstUser, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	firstRow.Status, firstRow.NextRunAt = constants.TopicPaused, nil
	if err := repo.UpdateTopic(ctx, &firstRow); err != nil {
		t.Fatal(err)
	}
	service := radarsvc.New(repo, integrationModel{}, integrationSearch{}, box)
	flowTopic, err := service.CreateTopic(ctx, firstUser, "关注模型发布")
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := service.RunDue(ctx, time.Now().UTC().Add(time.Second))
	if err != nil || !claimed {
		t.Fatalf("run due: claimed=%t err=%v", claimed, err)
	}
	flowFeed, err := service.ListFeed(ctx, firstUser, flowTopic.ID, 0, 20)
	if err != nil || len(flowFeed.Items) != 1 || flowFeed.Items[0].Title != "模型发布公告" {
		t.Fatalf("scan feed = %+v, %v", flowFeed, err)
	}
	flowRuns, err := service.ListRuns(ctx, firstUser, flowTopic.ID)
	if err != nil || len(flowRuns) != 1 || flowRuns[0].Status != constants.RunSucceeded {
		t.Fatalf("scan runs = %+v, %v", flowRuns, err)
	}
}
