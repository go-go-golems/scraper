# Changelog

## 2026-05-22

- Initial workspace created


## 2026-05-22

Investigated Hacker News 429 failures from SQLite databases; queue limiter appears to work, failures align with malformed page-3 URL.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/05/22/SCRAPER-HN-RATELIMIT-429--investigate-hacker-news-429-and-queue-rate-limiting/analysis/01-findings.md — Investigation findings


## 2026-05-22

Fixed Hacker News query-only pagination URL resolution so ?p=3 is resolved against the origin instead of appended to ?p=2; added regression test for three-page flow.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/site_test.go — Regression test proving page 3 fetch uses p=3
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/sites/hackernews/scripts/lib/frontpage.js — Fix query-only HN next-page URL handling
- not p — 2/?p=3

