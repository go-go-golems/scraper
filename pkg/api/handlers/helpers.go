package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	apitypes "github.com/go-go-golems/scraper/pkg/api/types"
	"github.com/go-go-golems/scraper/pkg/services/catalog"
	"github.com/go-go-golems/scraper/pkg/services/submission"
	"github.com/rs/zerolog/log"
)

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func decodeJSON(r *http.Request, dst interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, apitypes.ErrorResponse{
		Error: apitypes.APIError{
			Code:    code,
			Message: message,
		},
	})
}

func writeServiceError(w http.ResponseWriter, err error) {
	var notFound *catalog.NotFoundError
	var validation *submission.ValidationError
	switch {
	case errors.As(err, &notFound):
		writeError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.As(err, &validation):
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
	default:
		log.Error().Err(err).Msg("http api request failed")
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
