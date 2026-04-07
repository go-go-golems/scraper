# Changelog

## 2026-03-23

- Initial workspace created
- Converted Hacker News and Slashdot submission entrypoints to JS `verbs/*.js`
- Removed the bespoke Go `cli.go` and `workflow.go` files for both sites
- Updated the site definitions to expose `VerbsFS` and `VerbsRoot`
- Replaced the old inline-runner command tests with submit-plus-worker command-path tests
- Deleted the now-unused `pkg/sites/cliutil/http_runner.go` helper

- Ticket administratively closed on 2026-04-07 and retained as historical context; follow-on work should use newer focused tickets where they exist.
