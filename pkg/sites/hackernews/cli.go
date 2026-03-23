package hackernews

import (
	"github.com/go-go-golems/scraper/pkg/engine/model"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
	"github.com/go-go-golems/scraper/pkg/sites/cliutil"
	"github.com/spf13/cobra"
)

func registerCLI(root *cobra.Command) error {
	options := &cliutil.HTTPWorkflowCLIOptions{}

	siteCmd := &cobra.Command{
		Use:   "hackernews",
		Short: "Run built-in Hacker News workflows and operator smoke tests",
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run named Hacker News entrypoints",
	}

	seedCmd := &cobra.Command{
		Use:   "seed",
		Short: "Run the Hacker News seed workflow from seed.js",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliutil.RunHTTPWorkflowCommand(cmd, options, cliutil.HTTPWorkflowSpec{
				Site:           "hackernews",
				Entrypoint:     "seed",
				DefaultBaseURL: "https://news.ycombinator.com/",
				FixtureName:    "frontpage.html",
				BuildWorkflow: func(baseURL string, workflowID string, maxPages int) (params storecontract.CreateWorkflowParams, targetOpID model.OpID, err error) {
					return BuildSeedWorkflow(RunOptions{
						WorkflowID: workflowID,
						BaseURL:    baseURL,
						MaxPages:   maxPages,
					})
				},
				RegisterSite: Register,
				LoadFixture:  ReadFixture,
			})
		},
	}
	cliutil.AddSharedHTTPWorkflowFlags(seedCmd, options, "https://news.ycombinator.com/")

	extractCmd := &cobra.Command{
		Use:   "extract-frontpage",
		Short: "Run the Hacker News extract_frontpage.js stage with an explicit fetch dependency",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliutil.RunHTTPWorkflowCommand(cmd, options, cliutil.HTTPWorkflowSpec{
				Site:           "hackernews",
				Entrypoint:     "extract-frontpage",
				DefaultBaseURL: "https://news.ycombinator.com/",
				FixtureName:    "frontpage.html",
				BuildWorkflow: func(baseURL string, workflowID string, maxPages int) (params storecontract.CreateWorkflowParams, targetOpID model.OpID, err error) {
					return BuildExtractFrontpageWorkflow(RunOptions{
						WorkflowID: workflowID,
						BaseURL:    baseURL,
						MaxPages:   maxPages,
					})
				},
				RegisterSite: Register,
				LoadFixture:  ReadFixture,
			})
		},
	}
	cliutil.AddSharedHTTPWorkflowFlags(extractCmd, options, "https://news.ycombinator.com/")

	runCmd.AddCommand(seedCmd)
	runCmd.AddCommand(extractCmd)
	siteCmd.AddCommand(runCmd)
	root.AddCommand(siteCmd)
	return nil
}
