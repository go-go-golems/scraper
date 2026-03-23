package slashdot

import (
	"github.com/go-go-golems/scraper/pkg/engine/model"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
	"github.com/go-go-golems/scraper/pkg/sites/cliutil"
	"github.com/spf13/cobra"
)

func registerCLI(root *cobra.Command) error {
	options := &cliutil.HTTPWorkflowCLIOptions{}

	siteCmd := &cobra.Command{
		Use:   "slashdot",
		Short: "Run built-in Slashdot workflows and operator smoke tests",
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run named Slashdot entrypoints",
	}

	seedCmd := &cobra.Command{
		Use:   "seed",
		Short: "Run the Slashdot seed workflow from seed.js",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliutil.RunHTTPWorkflowCommand(cmd, options, cliutil.HTTPWorkflowSpec{
				Site:           "slashdot",
				Entrypoint:     "seed",
				DefaultBaseURL: "https://slashdot.org/",
				FixtureName:    "frontpage.html",
				BuildWorkflow: func(baseURL string, workflowID string) (params storecontract.CreateWorkflowParams, targetOpID model.OpID, err error) {
					return BuildSeedWorkflow(RunOptions{
						WorkflowID: workflowID,
						BaseURL:    baseURL,
					})
				},
				RegisterSite: Register,
				LoadFixture:  ReadFixture,
			})
		},
	}
	cliutil.AddSharedHTTPWorkflowFlags(seedCmd, options, "https://slashdot.org/")

	extractCmd := &cobra.Command{
		Use:   "extract-frontpage",
		Short: "Run the Slashdot extract_frontpage.js stage with an explicit fetch dependency",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliutil.RunHTTPWorkflowCommand(cmd, options, cliutil.HTTPWorkflowSpec{
				Site:           "slashdot",
				Entrypoint:     "extract-frontpage",
				DefaultBaseURL: "https://slashdot.org/",
				FixtureName:    "frontpage.html",
				BuildWorkflow: func(baseURL string, workflowID string) (params storecontract.CreateWorkflowParams, targetOpID model.OpID, err error) {
					return BuildExtractFrontpageWorkflow(RunOptions{
						WorkflowID: workflowID,
						BaseURL:    baseURL,
					})
				},
				RegisterSite: Register,
				LoadFixture:  ReadFixture,
			})
		},
	}
	cliutil.AddSharedHTTPWorkflowFlags(extractCmd, options, "https://slashdot.org/")

	runCmd.AddCommand(seedCmd)
	runCmd.AddCommand(extractCmd)
	siteCmd.AddCommand(runCmd)
	root.AddCommand(siteCmd)
	return nil
}
