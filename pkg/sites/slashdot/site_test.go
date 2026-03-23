package slashdot

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

func TestSlashdotFrontpageWorkflow(t *testing.T) {
	testSlashdotWorkflow(t, "seed", func(baseURL string) (storecontract.CreateWorkflowParams, model.OpID, error) {
		return BuildSeedWorkflow(RunOptions{
			WorkflowID: "wf-slashdot-seed",
			BaseURL:    baseURL,
		})
	})
}

func TestSlashdotExtractFrontpageWorkflow(t *testing.T) {
	testSlashdotWorkflow(t, "extract-frontpage", func(baseURL string) (storecontract.CreateWorkflowParams, model.OpID, error) {
		return BuildExtractFrontpageWorkflow(RunOptions{
			WorkflowID: "wf-slashdot-extract",
			BaseURL:    baseURL,
		})
	})
}

func TestSlashdotSeedWorkflowFollowsPagination(t *testing.T) {
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

	pageOne := slashdotFixturePage(
		[]slashdotFixtureStory{
			{ID: "181087690", Title: "Mark Zuckerberg Is Building an AI Agent To Help Him Be CEO", StoryURL: "//tech.slashdot.org/story/26/03/23/1900208/mark-zuckerberg-is-building-an-ai-agent-to-help-him-be-ceo", SourceName: "the-independent.com", SourceURL: "https://www.the-independent.com/example", CommentsURL: "//tech.slashdot.org/story/26/03/23/1900208/mark-zuckerberg-is-building-an-ai-agent-to-help-him-be-ceo#comments", CommentsCount: 1, Author: "BeauHD", Department: "good-luck-with-that", PostedAtText: "on Monday March 23, 2026 @02:00PM"},
			{ID: "181087016", Title: "Space-Based Solar Power Study Progresses", StoryURL: "//science.slashdot.org/story/26/03/23/1715239/space-based-solar-power-study-progresses", SourceName: "nasa.gov", SourceURL: "https://www.nasa.gov/example", CommentsURL: "//science.slashdot.org/story/26/03/23/1715239/space-based-solar-power-study-progresses#comments", CommentsCount: 24, Author: "feedme", Department: "beam-me-up", PostedAtText: "on Monday March 23, 2026 @11:30AM"},
		},
		"?page=1",
	)
	pageTwo := slashdotFixturePage(
		[]slashdotFixtureStory{
			{ID: "181086000", Title: "Older Link Chain Story One", StoryURL: "//developers.slashdot.org/story/26/03/23/1600000/older-link-chain-story-one", SourceName: "example.net", SourceURL: "https://example.net/story-one", CommentsURL: "//developers.slashdot.org/story/26/03/23/1600000/older-link-chain-story-one#comments", CommentsCount: 7, Author: "olderone", Department: "page-two", PostedAtText: "on Monday March 23, 2026 @10:15AM"},
			{ID: "181085999", Title: "Older Link Chain Story Two", StoryURL: "//hardware.slashdot.org/story/26/03/23/1500000/older-link-chain-story-two", SourceName: "example.org", SourceURL: "https://example.org/story-two", CommentsURL: "//hardware.slashdot.org/story/26/03/23/1500000/older-link-chain-story-two#comments", CommentsCount: 3, Author: "oldertwo", Department: "page-two-again", PostedAtText: "on Monday March 23, 2026 @09:45AM"},
		},
		"",
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.RawQuery {
		case "":
			_, _ = w.Write([]byte(pageOne))
		case "page=1":
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
	}, "worker-slashdot", nil)
	require.NoError(t, err)
	s.SetSiteDBProvider(func(ctx context.Context, site model.SiteName) (databasemod.QueryExecer, error) {
		require.Equal(t, model.SiteName("slashdot"), site)
		return siteDB, nil
	})

	params, _, err := BuildSeedWorkflow(RunOptions{
		WorkflowID: "wf-slashdot-seed-multipage",
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

	rows, err := siteDB.Query(`SELECT story_id FROM stories ORDER BY story_id`)
	require.NoError(t, err)
	defer rows.Close()

	var gotIDs []string
	for rows.Next() {
		var id string
		require.NoError(t, rows.Scan(&id))
		gotIDs = append(gotIDs, id)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []string{
		"181085999",
		"181086000",
		"181087016",
		"181087690",
	}, gotIDs)
}

func testSlashdotWorkflow(t *testing.T, _ string, build func(baseURL string) (storecontract.CreateWorkflowParams, model.OpID, error)) {
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

type slashdotFixtureStory struct {
	ID            string
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

func slashdotFixturePage(stories []slashdotFixtureStory, nextHref string) string {
	page := "<html><body><section id=\"firehose\">"
	for _, story := range stories {
		page += fmt.Sprintf(
			"<article id=\"firehose-%s\" data-fhid=\"%s\" data-fhtype=\"story\" class=\"fhitem fhitem-story article\">"+
				"<header>"+
				"<span id=\"title-%s\" class=\"story-title\">"+
				"<a href=\"%s\">%s</a>"+
				"<span class=\"no extlnk\"><a class=\"story-sourcelnk\" href=\"%s\" title=\"\"> (%s) </a></span>"+
				"</span>"+
				"<span class=\"comment-bubble\"><a href=\"%s\" title=\"\">%d</a></span>"+
				"</header>"+
				"<div class=\"details\">Posted by <a href=\"//slashdot.org/~%s\">%s</a>"+
				"<time id=\"fhtime-%s\" datetime=\"%s\">%s</time>"+
				" from the <span class=\"dept-text\">%s</span> dept.</div>"+
				"</article>",
			story.ID,
			story.ID,
			story.ID,
			story.StoryURL,
			story.Title,
			story.SourceURL,
			story.SourceName,
			story.CommentsURL,
			story.CommentsCount,
			story.Author,
			story.Author,
			story.ID,
			story.PostedAtText,
			story.PostedAtText,
			story.Department,
		)
	}
	page += "</section>"
	if nextHref != "" {
		page += fmt.Sprintf("<a class=\"prevnextbutact\" href=\"%s\">Older &raquo;</a>", nextHref)
	}
	page += "</body></html>"
	return page
}
