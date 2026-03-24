package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-go-golems/scraper/pkg/sites/defaults"
	"github.com/stretchr/testify/require"
)

func TestServerHealthAndCatalogEndpoints(t *testing.T) {
	registry, err := defaults.NewRegistry()
	require.NoError(t, err)

	server := New(Config{
		Address:      "127.0.0.1:0",
		EngineDB:     t.TempDir() + "/engine.db",
		SitesDir:     t.TempDir(),
		ReadTimeout:  5,
		WriteTimeout: 5,
		Version:      "test-version",
	}, registry)

	ts := httptest.NewServer(server.Handler)
	defer ts.Close()

	t.Run("healthz", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/healthz")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		payload := map[string]bool{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
		require.Equal(t, true, payload["ok"])
	})

	t.Run("sites", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/sites")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		payload := struct {
			Sites []struct {
				Name string `json:"name"`
			} `json:"sites"`
		}{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
		require.NotEmpty(t, payload.Sites)
	})

	t.Run("verb details", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/sites/js-demo/verbs/seed")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		payload := struct {
			Verb struct {
				Name        string `json:"name"`
				CommandPath string `json:"commandPath"`
				SourceFile  string `json:"sourceFile"`
			} `json:"verb"`
		}{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
		require.Equal(t, "seed", payload.Verb.Name)
		require.Equal(t, "site js-demo run seed", payload.Verb.CommandPath)
		require.Equal(t, "seed.js", payload.Verb.SourceFile)
	})
}
