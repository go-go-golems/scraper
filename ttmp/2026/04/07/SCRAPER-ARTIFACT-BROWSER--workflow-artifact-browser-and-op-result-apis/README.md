# Workflow artifact browser and op result APIs

This is the document workspace for ticket SCRAPER-ARTIFACT-BROWSER.

## Structure

- **design/**: Design documents and architecture notes
- **reference/**: Reference documentation and API contracts
- **playbooks/**: Operational playbooks and procedures
- **scripts/**: Utility scripts and automation
- **sources/**: External sources and imported documents
- **various/**: Scratch or meeting notes, working notes
- **archive/**: Optional space for deprecated or reference-only artifacts

## Design documents

- **[01-artifact-browser-and-op-result-implementation-guide.md](design/01-artifact-browser-and-op-result-implementation-guide.md)** — Backend implementation guide: API contracts, service methods, handler design, DTOs, and test plan. Audience: intern implementing the backend.
- **[02-artifact-browser-frontend-ui-design.md](design/02-artifact-browser-frontend-ui-design.md)** — Full frontend UI design: ASCII screen mockups, YAML component DSL, API-to-component mapping, new RTK Query endpoint definition, and step-by-step implementation order. Audience: engineer implementing Phase 2 (frontend).

## Getting Started

Use docmgr commands to manage this workspace:

- Add documents: `docmgr doc add --ticket SCRAPER-ARTIFACT-BROWSER --doc-type design-doc --title "My Design"`
- Import sources: `docmgr import file --ticket SCRAPER-ARTIFACT-BROWSER --file /path/to/doc.md`
- Update metadata: `docmgr meta update --ticket SCRAPER-ARTIFACT-BROWSER --field Status --value review`
