# Changelog

## 2026-04-07

- Initial workspace created
- Added the orchestration cleanup design and task plan for the API server and worker command files.
- Split API server route registration by domain into dedicated route files while keeping `server.New(...)` as the composition root.
