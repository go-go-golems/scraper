package handlers

import (
	"net/http"

	apitypes "github.com/go-go-golems/scraper/pkg/api/types"
	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/services/catalog"
)

type CatalogHandler struct {
	service  *catalog.Service
	version  string
	address  string
	engineDB string
	sitesDir string
}

func NewCatalogHandler(service *catalog.Service, version string, address string, engineDB string, sitesDir string) *CatalogHandler {
	return &CatalogHandler{
		service:  service,
		version:  version,
		address:  address,
		engineDB: engineDB,
		sitesDir: sitesDir,
	}
}

func (h *CatalogHandler) Healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *CatalogHandler) Info(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, apitypes.InfoResponse{
		Version:  h.version,
		Address:  h.address,
		EngineDB: h.engineDB,
		SitesDir: h.sitesDir,
		Sites:    h.service.ListSites(),
		Now:      nowUTC(),
	})
}

func (h *CatalogHandler) Sites(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"sites": h.service.ListSites()})
}

func (h *CatalogHandler) Site(w http.ResponseWriter, r *http.Request) {
	site, _, err := h.service.GetSite(model.SiteName(r.PathValue("site")))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"site": site})
}

func (h *CatalogHandler) Verbs(w http.ResponseWriter, r *http.Request) {
	verbs, err := h.service.ListVerbs(model.SiteName(r.PathValue("site")))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"verbs": verbs})
}

func (h *CatalogHandler) Verb(w http.ResponseWriter, r *http.Request) {
	verb, _, _, _, err := h.service.GetVerb(model.SiteName(r.PathValue("site")), r.PathValue("verb"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"verb": verb})
}

func (h *CatalogHandler) SiteDetail(w http.ResponseWriter, r *http.Request) {
	detail, err := h.service.GetSiteDetail(model.SiteName(r.PathValue("site")))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"site": detail})
}

func (h *CatalogHandler) Scripts(w http.ResponseWriter, r *http.Request) {
	scripts, err := h.service.ListScripts(model.SiteName(r.PathValue("site")))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, apitypes.ScriptListResponse{Scripts: scripts})
}

func (h *CatalogHandler) Script(w http.ResponseWriter, r *http.Request) {
	site := model.SiteName(r.PathValue("site"))
	scriptPath := r.PathValue("path")
	source, err := h.service.ReadScript(site, scriptPath)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, apitypes.ScriptResponse{Path: scriptPath, Source: source})
}
