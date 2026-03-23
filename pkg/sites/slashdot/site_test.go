package slashdot

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

func TestSlashdotFrontpageWorkflow(t *testing.T) {
	ctx := context.Background()
	registry := siteregistry.New()
	require.NoError(t, Register(registry))

	sitesDir := t.TempDir()
	manager := sitemigrate.NewManager(registry)
	report, err := manager.Migrate(ctx, model.SiteName("slashdot"), sitesDir)
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
	}, "worker-slashdot", nil)
	require.NoError(t, err)
	s.SetSiteDBProvider(func(ctx context.Context, site model.SiteName) (databasemod.QueryExecer, error) {
		require.Equal(t, model.SiteName("slashdot"), site)
		return siteDB, nil
	})

	baseURL := server.URL + "/"
	input, err := json.Marshal(map[string]any{"baseURL": baseURL})
	require.NoError(t, err)

	err = s.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{
		Workflow: model.WorkflowRun{
			ID:   "wf-slashdot",
			Site: "slashdot",
			Name: "slashdot frontpage",
		},
		Initial: []model.OpSpec{
			{
				ID:         "seed-slashdot",
				WorkflowID: "wf-slashdot",
				Site:       "slashdot",
				Kind:       "js",
				Queue:      "site:slashdot:js",
				DedupKey:   "seed-slashdot",
				Input:      input,
				Metadata:   map[string]string{"script": "seed.js"},
			},
		},
	})
	require.NoError(t, err)

	for i := 0; i < 8; i++ {
		_, err = s.RunOnce(ctx)
		require.NoError(t, err)

		workflow, err := engineStore.GetWorkflow(ctx, "wf-slashdot")
		require.NoError(t, err)
		require.NotNil(t, workflow)
		if workflow.Status == model.WorkflowStatusSucceeded {
			break
		}
		require.NotEqual(t, model.WorkflowStatusFailed, workflow.Status)
	}

	workflow, err := engineStore.GetWorkflow(ctx, "wf-slashdot")
	require.NoError(t, err)
	require.Equal(t, model.WorkflowStatusSucceeded, workflow.Status)

	rows, err := siteDB.Query(`
		SELECT story_id, position, title, story_url, source_name, source_url, comments_url, comments_count, author, department, posted_at_text
		FROM stories
		ORDER BY position
	`)
	require.NoError(t, err)
	defer rows.Close()

	type storyRow struct {
		ID            string
		Position      int
		Title         string
		StoryURL      string
		SourceName    string
		SourceURL     string
		CommentsURL   string
		CommentsCount int
		Author        string
		Department    string
		PostedAtText  string
	}

	var got []storyRow
	for rows.Next() {
		var row storyRow
		require.NoError(t, rows.Scan(
			&row.ID,
			&row.Position,
			&row.Title,
			&row.StoryURL,
			&row.SourceName,
			&row.SourceURL,
			&row.CommentsURL,
			&row.CommentsCount,
			&row.Author,
			&row.Department,
			&row.PostedAtText,
		))
		got = append(got, row)
	}
	require.NoError(t, rows.Err())
	require.Len(t, got, 2)

	require.Equal(t, storyRow{
		ID:            "181087690",
		Position:      1,
		Title:         "Mark Zuckerberg Is Building an AI Agent To Help Him Be CEO",
		StoryURL:      "https://tech.slashdot.org/story/26/03/23/1900208/mark-zuckerberg-is-building-an-ai-agent-to-help-him-be-ceo",
		SourceName:    "the-independent.com",
		SourceURL:     "https://www.the-independent.com/example",
		CommentsURL:   "https://tech.slashdot.org/story/26/03/23/1900208/mark-zuckerberg-is-building-an-ai-agent-to-help-him-be-ceo#comments",
		CommentsCount: 1,
		Author:        "BeauHD",
		Department:    "good-luck-with-that",
		PostedAtText:  "on Monday March 23, 2026 @02:00PM",
	}, got[0])

	require.Equal(t, storyRow{
		ID:            "181087016",
		Position:      2,
		Title:         "Space-Based Solar Power Study Progresses",
		StoryURL:      "https://science.slashdot.org/story/26/03/23/1715239/space-based-solar-power-study-progresses",
		SourceName:    "nasa.gov",
		SourceURL:     "https://www.nasa.gov/example",
		CommentsURL:   "https://science.slashdot.org/story/26/03/23/1715239/space-based-solar-power-study-progresses#comments",
		CommentsCount: 24,
		Author:        "feedme",
		Department:    "beam-me-up",
		PostedAtText:  "on Monday March 23, 2026 @11:30AM",
	}, got[1])
}
