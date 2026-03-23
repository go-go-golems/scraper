package hackernews

import (
	"context"
	"database/sql"
	"fmt"
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
	testHackerNewsWorkflow(t, "seed", func(baseURL string) (storecontract.CreateWorkflowParams, model.OpID, error) {
		return BuildSeedWorkflow(RunOptions{
			WorkflowID: "wf-hackernews-seed",
			BaseURL:    baseURL,
		})
	})
}

func TestHackerNewsExtractFrontpageWorkflow(t *testing.T) {
	testHackerNewsWorkflow(t, "extract-frontpage", func(baseURL string) (storecontract.CreateWorkflowParams, model.OpID, error) {
		return BuildExtractFrontpageWorkflow(RunOptions{
			WorkflowID: "wf-hackernews-extract",
			BaseURL:    baseURL,
		})
	})
}

func TestHackerNewsSeedWorkflowFollowsPagination(t *testing.T) {
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

	pageOne := hackerNewsFixturePage(
		[]hackerNewsFixtureStory{
			{ID: "47490070", Rank: 1, Title: "First & Strong Story", StoryURL: "https://example.com/first-story", SiteName: "example.com", Score: 236, Author: "anemll", AgeText: "3 hours ago", CommentsText: "138 comments"},
			{ID: "47490080", Rank: 2, Title: "Ask HN: Building Smaller Scrapers?", StoryURL: "item?id=47490080", SiteName: "", Score: 87, Author: "somebody", AgeText: "2 hours ago", CommentsText: "discuss"},
		},
		"?p=2",
	)
	pageTwo := hackerNewsFixturePage(
		[]hackerNewsFixtureStory{
			{ID: "47490100", Rank: 31, Title: "A Deeper Page Story", StoryURL: "https://example.com/deeper-story", SiteName: "example.com", Score: 55, Author: "deeper", AgeText: "1 hour ago", CommentsText: "12 comments"},
			{ID: "47490110", Rank: 32, Title: "Ask HN: Page Two", StoryURL: "item?id=47490110", SiteName: "", Score: 21, Author: "pager", AgeText: "50 minutes ago", CommentsText: "5 comments"},
		},
		"",
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.RawQuery {
		case "":
			_, _ = w.Write([]byte(pageOne))
		case "p=2":
			_, _ = w.Write([]byte(pageTwo))
		default:
			http.NotFound(w, r)
		}
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

	params, _, err := BuildSeedWorkflow(RunOptions{
		WorkflowID: "wf-hackernews-seed-multipage",
		BaseURL:    server.URL + "/",
		MaxPages:   2,
	})
	require.NoError(t, err)
	require.NoError(t, s.CreateWorkflow(ctx, params))

	for i := 0; i < 16; i++ {
		_, err = s.RunOnce(ctx)
		require.NoError(t, err)

		workflow, err := engineStore.GetWorkflow(ctx, params.Workflow.ID)
		require.NoError(t, err)
		require.NotNil(t, workflow)
		if workflow.Status == model.WorkflowStatusSucceeded {
			break
		}
		require.NotEqual(t, model.WorkflowStatusFailed, workflow.Status)
	}

	workflow, err := engineStore.GetWorkflow(ctx, params.Workflow.ID)
	require.NoError(t, err)
	require.Equal(t, model.WorkflowStatusSucceeded, workflow.Status)

	rows, err := siteDB.Query(`SELECT story_id, rank FROM stories ORDER BY rank`)
	require.NoError(t, err)
	defer rows.Close()

	var gotIDs []string
	for rows.Next() {
		var id string
		var rank int
		require.NoError(t, rows.Scan(&id, &rank))
		gotIDs = append(gotIDs, fmt.Sprintf("%d:%s", rank, id))
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []string{
		"1:47490070",
		"2:47490080",
		"31:47490100",
		"32:47490110",
	}, gotIDs)
}

func testHackerNewsWorkflow(t *testing.T, _ string, build func(baseURL string) (storecontract.CreateWorkflowParams, model.OpID, error)) {
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
	params, _, err := build(baseURL)
	require.NoError(t, err)

	err = s.CreateWorkflow(ctx, params)
	require.NoError(t, err)

	for i := 0; i < 8; i++ {
		_, err = s.RunOnce(ctx)
		require.NoError(t, err)

		workflow, err := engineStore.GetWorkflow(ctx, params.Workflow.ID)
		require.NoError(t, err)
		require.NotNil(t, workflow)
		if workflow.Status == model.WorkflowStatusSucceeded {
			break
		}
		require.NotEqual(t, model.WorkflowStatusFailed, workflow.Status)
	}

	workflow, err := engineStore.GetWorkflow(ctx, params.Workflow.ID)
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

type hackerNewsFixtureStory struct {
	ID           string
	Rank         int
	Title        string
	StoryURL     string
	SiteName     string
	Score        int
	Author       string
	AgeText      string
	CommentsText string
}

func hackerNewsFixturePage(stories []hackerNewsFixtureStory, nextHref string) string {
	page := "<html><body><table class=\"itemlist\">"
	for _, story := range stories {
		siteMarkup := ""
		if story.SiteName != "" {
			siteMarkup = fmt.Sprintf(
				"<span class=\"sitebit comhead\"> (<a href=\"from?site=%s\"><span class=\"sitestr\">%s</span></a>) </span>",
				story.SiteName,
				story.SiteName,
			)
		}
		page += fmt.Sprintf(
			"<tr class=\"athing submission\" id=\"%s\">"+
				"<td class=\"title\"><span class=\"rank\">%d.</span></td>"+
				"<td class=\"title\"><span class=\"titleline\"><a href=\"%s\">%s</a>%s</span></td>"+
				"</tr>"+
				"<tr><td colspan=\"2\"></td><td class=\"subtext\">"+
				"<span class=\"score\" id=\"score_%s\">%d points</span> by "+
				"<a href=\"user?id=%s\" class=\"hnuser\">%s</a> "+
				"<span class=\"age\" title=\"2026-03-23T10:00:00 1742724000\"><a href=\"item?id=%s\">%s</a></span> "+
				"<span id=\"unv_%s\"></span> | <a href=\"item?id=%s\">%s</a>"+
				"</td></tr>",
			story.ID,
			story.Rank,
			story.StoryURL,
			story.Title,
			siteMarkup,
			story.ID,
			story.Score,
			story.Author,
			story.Author,
			story.ID,
			story.AgeText,
			story.ID,
			story.ID,
			story.CommentsText,
		)
	}
	if nextHref != "" {
		page += fmt.Sprintf("<tr><td colspan=\"2\"></td><td class='title'><a href='%s' class='morelink' rel='next'>More</a></td></tr>", nextHref)
	}
	page += "</table></body></html>"
	return page
}
