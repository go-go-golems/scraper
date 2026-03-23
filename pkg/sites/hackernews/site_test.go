package hackernews

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	databasemod "github.com/go-go-golems/go-go-goja/modules/database"
	"github.com/go-go-golems/scraper/pkg/engine/config"
	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/engine/runner"
	"github.com/go-go-golems/scraper/pkg/engine/scheduler"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
	sqlitestore "github.com/go-go-golems/scraper/pkg/engine/store/sqlite"
	sitemigrate "github.com/go-go-golems/scraper/pkg/sites/migrate"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestHackerNewsFrontpageWorkflow(t *testing.T) {
	ctx := context.Background()
	registry := siteregistry.New()
	require.NoError(t, Register(registry))

	sitesDir := t.TempDir()
	manager := sitemigrate.NewManager(registry)
	report, err := manager.Migrate(ctx, model.SiteName("hackernews"), sitesDir)
	require.NoError(t, err)

	siteDB, err := sql.Open("sqlite3", report.DatabasePath)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, siteDB.Close()) })

	engineStore, err := sqlitestore.Open(ctx, filepath.Join(t.TempDir(), "engine.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, engineStore.Close()) })

	fixture, err := ReadFixture("frontpage.html")
	require.NoError(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(fixture)
	}))
	t.Cleanup(server.Close)

	runners := runner.NewRegistry()
	require.NoError(t, runners.Register(runner.NewHTTPRunner(config.HTTP{
		UserAgent: "scraper/test",
		Timeout:   5 * time.Second,
	}, server.Client())))
	require.NoError(t, runners.Register(runner.NewJSRunner(registry)))

	s, err := scheduler.New(engineStore, runners, scheduler.Config{
		MaxWorkers:           4,
		PollInterval:         100 * time.Millisecond,
		DefaultLeaseDuration: 30 * time.Second,
	}, "worker-hn", nil)
	require.NoError(t, err)
	s.SetSiteDBProvider(func(ctx context.Context, site model.SiteName) (databasemod.QueryExecer, error) {
		require.Equal(t, model.SiteName("hackernews"), site)
		return siteDB, nil
	})

	baseURL := server.URL + "/"
	input, err := json.Marshal(map[string]any{"baseURL": baseURL})
	require.NoError(t, err)

	err = s.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{
		Workflow: model.WorkflowRun{
			ID:   "wf-hackernews",
			Site: "hackernews",
			Name: "hackernews frontpage",
		},
		Initial: []model.OpSpec{
			{
				ID:         "seed-hackernews",
				WorkflowID: "wf-hackernews",
				Site:       "hackernews",
				Kind:       "js",
				Queue:      "site:hackernews:js",
				DedupKey:   "seed-hackernews",
				Input:      input,
				Metadata:   map[string]string{"script": "seed.js"},
			},
		},
	})
	require.NoError(t, err)

	for i := 0; i < 8; i++ {
		_, err = s.RunOnce(ctx)
		require.NoError(t, err)

		workflow, err := engineStore.GetWorkflow(ctx, "wf-hackernews")
		require.NoError(t, err)
		require.NotNil(t, workflow)
		if workflow.Status == model.WorkflowStatusSucceeded {
			break
		}
		require.NotEqual(t, model.WorkflowStatusFailed, workflow.Status)
	}

	workflow, err := engineStore.GetWorkflow(ctx, "wf-hackernews")
	require.NoError(t, err)
	require.Equal(t, model.WorkflowStatusSucceeded, workflow.Status)

	rows, err := siteDB.Query(`
		SELECT story_id, rank, title, story_url, site_name, score, author, age_text, comments_url, comments_count
		FROM stories
		ORDER BY rank
	`)
	require.NoError(t, err)
	defer rows.Close()

	type storyRow struct {
		ID            string
		Rank          int
		Title         string
		StoryURL      string
		SiteName      string
		Score         int
		Author        string
		AgeText       string
		CommentsURL   string
		CommentsCount int
	}

	var got []storyRow
	for rows.Next() {
		var row storyRow
		require.NoError(t, rows.Scan(
			&row.ID,
			&row.Rank,
			&row.Title,
			&row.StoryURL,
			&row.SiteName,
			&row.Score,
			&row.Author,
			&row.AgeText,
			&row.CommentsURL,
			&row.CommentsCount,
		))
		got = append(got, row)
	}
	require.NoError(t, rows.Err())
	require.Len(t, got, 2)

	require.Equal(t, storyRow{
		ID:            "47490070",
		Rank:          1,
		Title:         "First & Strong Story",
		StoryURL:      "https://example.com/first-story",
		SiteName:      "example.com",
		Score:         236,
		Author:        "anemll",
		AgeText:       "3 hours ago",
		CommentsURL:   baseURL + "item?id=47490070",
		CommentsCount: 138,
	}, got[0])

	require.Equal(t, storyRow{
		ID:            "47490080",
		Rank:          2,
		Title:         "Ask HN: Building Smaller Scrapers?",
		StoryURL:      baseURL + "item?id=47490080",
		SiteName:      "",
		Score:         87,
		Author:        "somebody",
		AgeText:       "2 hours ago",
		CommentsURL:   baseURL + "item?id=47490080",
		CommentsCount: 0,
	}, got[1])
}
