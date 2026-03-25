package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/config"
	"github.com/go-go-golems/scraper/pkg/engine/model"
)

type HTTPRunner struct {
	client *http.Client
	config config.HTTP
}

func NewHTTPRunner(cfg config.HTTP, client *http.Client) (*HTTPRunner, error) {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 15 * time.Second
	}
	explicitProxyURL, err := parseExplicitProxyURL(cfg.ProxyURL)
	if err != nil {
		return nil, err
	}
	if client == nil {
		client = &http.Client{
			Timeout: cfg.Timeout,
		}
	} else if client.Timeout <= 0 {
		client.Timeout = cfg.Timeout
	}
	if explicitProxyURL != nil {
		transport, ok := http.DefaultTransport.(*http.Transport)
		if !ok {
			return nil, fmt.Errorf("unexpected default transport type %T", http.DefaultTransport)
		}
		cloned := transport.Clone()
		cloned.Proxy = http.ProxyURL(explicitProxyURL)
		client.Transport = cloned
	}

	return &HTTPRunner{
		client: client,
		config: cfg,
	}, nil
}

func (r *HTTPRunner) Kind() string {
	return "http/fetch"
}

type httpFetchInput struct {
	Request      httpRequestSpec `json:"request"`
	PersistBody  bool            `json:"persistBody"`
	ArtifactName string          `json:"artifactName"`
}

type httpRequestSpec struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Form    map[string]string `json:"form"`
	Body    string            `json:"body"`
}

type httpExecutionEnvelope struct {
	Request  httpRenderedRequest  `json:"request"`
	Response httpResponseEnvelope `json:"response"`
}

type httpRenderedRequest struct {
	Method      string              `json:"method"`
	URL         string              `json:"url"`
	Headers     map[string][]string `json:"headers"`
	BodyText    string              `json:"bodyText,omitempty"`
	ContentType string              `json:"contentType,omitempty"`
}

type httpResponseEnvelope struct {
	StatusCode     int                 `json:"statusCode"`
	Status         string              `json:"status"`
	FinalURL       string              `json:"finalURL"`
	Headers        map[string][]string `json:"headers"`
	ContentType    string              `json:"contentType,omitempty"`
	ContentLength  int64               `json:"contentLength"`
	BodyArtifactID string              `json:"bodyArtifactID,omitempty"`
}

func (r *HTTPRunner) Run(ctx context.Context, runCtx RunContext) (*model.OpResult, error) {
	requestInput, templateData, err := decodeHTTPFetchInput(runCtx)
	if err != nil {
		return nil, err
	}

	renderedReq, bodyBytes, contentType, err := renderHTTPRequest(requestInput.Request, templateData)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, renderedReq.Method, renderedReq.URL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, runnerError("invalid_request", err.Error(), false, nil, runCtx.Now)
	}
	for key, values := range renderedReq.Headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	if req.Header.Get("User-Agent") == "" && strings.TrimSpace(r.config.UserAgent) != "" {
		req.Header.Set("User-Agent", r.config.UserAgent)
	}
	if req.Header.Get("Content-Type") == "" && contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	renderedReq.Headers = cloneHeaders(req.Header)
	renderedReq.ContentType = req.Header.Get("Content-Type")

	response, err := r.client.Do(req)
	if err != nil {
		return &model.OpResult{
			OpID: runCtx.Op.ID,
			Error: &model.OpError{
				Code:       "transport_error",
				Message:    err.Error(),
				Retryable:  true,
				OccurredAt: runCtx.Now,
			},
			CompletedAt: runCtx.Now,
		}, nil
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return &model.OpResult{
			OpID: runCtx.Op.ID,
			Error: &model.OpError{
				Code:       "read_response_error",
				Message:    err.Error(),
				Retryable:  true,
				OccurredAt: runCtx.Now,
			},
			CompletedAt: runCtx.Now,
		}, nil
	}

	resultEnvelope := httpExecutionEnvelope{
		Request: renderedReq,
		Response: httpResponseEnvelope{
			StatusCode:    response.StatusCode,
			Status:        response.Status,
			FinalURL:      response.Request.URL.String(),
			Headers:       cloneHeaders(response.Header),
			ContentType:   response.Header.Get("Content-Type"),
			ContentLength: int64(len(responseBody)),
		},
	}

	result := &model.OpResult{
		OpID:        runCtx.Op.ID,
		CompletedAt: runCtx.Now,
	}

	if requestInput.PersistBody {
		artifactName := strings.TrimSpace(requestInput.ArtifactName)
		if artifactName == "" {
			artifactName = "response-body"
		}
		artifactID := model.ArtifactID(fmt.Sprintf("%s:response-body", runCtx.Op.ID))
		result.Artifacts = append(result.Artifacts, model.ArtifactWrite{
			ID:          artifactID,
			Name:        artifactName,
			Kind:        "http-response-body",
			ContentType: bodyContentType(response.Header.Get("Content-Type")),
			Metadata: map[string]string{
				"method":      renderedReq.Method,
				"url":         renderedReq.URL,
				"status_code": fmt.Sprintf("%d", response.StatusCode),
			},
			Body: responseBody,
		})
		resultEnvelope.Response.BodyArtifactID = string(artifactID)
	}

	result.Data, err = json.Marshal(resultEnvelope)
	if err != nil {
		return nil, fmt.Errorf("marshal http result envelope: %w", err)
	}

	if response.StatusCode >= 400 {
		result.Error = &model.OpError{
			Code:       statusErrorCode(response.StatusCode),
			Message:    fmt.Sprintf("http request failed with status %d", response.StatusCode),
			Retryable:  retryableStatus(response.StatusCode),
			Details:    mustMarshalRaw(resultEnvelope),
			OccurredAt: runCtx.Now,
		}
	}

	return result, nil
}

func decodeHTTPFetchInput(runCtx RunContext) (*httpFetchInput, map[string]any, error) {
	payload := &httpFetchInput{}
	if len(runCtx.Op.Input) == 0 {
		return nil, nil, runnerError("invalid_input", "http/fetch op input is required", false, nil, runCtx.Now)
	}
	if err := json.Unmarshal(runCtx.Op.Input, payload); err != nil {
		return nil, nil, runnerError("invalid_input", fmt.Sprintf("decode http input: %v", err), false, nil, runCtx.Now)
	}
	if strings.TrimSpace(payload.Request.URL) == "" {
		return nil, nil, runnerError("invalid_input", "http request url is required", false, nil, runCtx.Now)
	}

	templateInput := map[string]any{}
	if err := json.Unmarshal(runCtx.Op.Input, &templateInput); err != nil {
		return nil, nil, runnerError("invalid_input", fmt.Sprintf("decode template input: %v", err), false, nil, runCtx.Now)
	}

	workflowInput := map[string]any{}
	if len(runCtx.Workflow.Input) > 0 {
		if err := json.Unmarshal(runCtx.Workflow.Input, &workflowInput); err != nil {
			return nil, nil, runnerError("invalid_input", fmt.Sprintf("decode workflow input: %v", err), false, nil, runCtx.Now)
		}
	}

	return payload, map[string]any{
		"input":    templateInput,
		"workflow": map[string]any{"input": workflowInput},
		"op": map[string]any{
			"id":       runCtx.Op.ID,
			"site":     runCtx.Op.Site,
			"kind":     runCtx.Op.Kind,
			"queue":    runCtx.Op.Queue,
			"dedupKey": runCtx.Op.DedupKey,
			"metadata": runCtx.Op.Metadata,
		},
	}, nil
}

func parseExplicitProxyURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	proxyURL, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse explicit proxy url: %w", err)
	}
	if proxyURL.Scheme == "" || proxyURL.Host == "" {
		return nil, fmt.Errorf("parse explicit proxy url: missing scheme or host")
	}
	return proxyURL, nil
}

func renderHTTPRequest(spec httpRequestSpec, templateData map[string]any) (httpRenderedRequest, []byte, string, error) {
	method := strings.ToUpper(strings.TrimSpace(spec.Method))
	if method == "" {
		method = http.MethodGet
	}

	renderedURL, err := renderTemplateString(spec.URL, templateData)
	if err != nil {
		return httpRenderedRequest{}, nil, "", runnerError("template_error", fmt.Sprintf("render request url: %v", err), false, nil, time.Time{})
	}

	headers := map[string][]string{}
	for key, value := range spec.Headers {
		renderedValue, err := renderTemplateString(value, templateData)
		if err != nil {
			return httpRenderedRequest{}, nil, "", runnerError("template_error", fmt.Sprintf("render header %s: %v", key, err), false, nil, time.Time{})
		}
		headers[key] = append(headers[key], renderedValue)
	}

	var bodyBytes []byte
	var bodyText string
	contentType := ""
	switch {
	case len(spec.Form) > 0:
		form := url.Values{}
		for key, value := range spec.Form {
			renderedValue, err := renderTemplateString(value, templateData)
			if err != nil {
				return httpRenderedRequest{}, nil, "", runnerError("template_error", fmt.Sprintf("render form field %s: %v", key, err), false, nil, time.Time{})
			}
			form.Set(key, renderedValue)
		}
		bodyText = form.Encode()
		bodyBytes = []byte(bodyText)
		contentType = "application/x-www-form-urlencoded"
	case spec.Body != "":
		bodyText, err = renderTemplateString(spec.Body, templateData)
		if err != nil {
			return httpRenderedRequest{}, nil, "", runnerError("template_error", fmt.Sprintf("render request body: %v", err), false, nil, time.Time{})
		}
		bodyBytes = []byte(bodyText)
	}

	return httpRenderedRequest{
		Method:      method,
		URL:         renderedURL,
		Headers:     headers,
		BodyText:    bodyText,
		ContentType: contentType,
	}, bodyBytes, contentType, nil
}

func renderTemplateString(source string, data map[string]any) (string, error) {
	tpl, err := template.New("http-runner").Option("missingkey=error").Parse(source)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func cloneHeaders(headers http.Header) map[string][]string {
	ret := map[string][]string{}
	for key, values := range headers {
		ret[key] = append([]string(nil), values...)
	}
	return ret
}

func statusErrorCode(statusCode int) string {
	return fmt.Sprintf("http_status_%d", statusCode)
}

func retryableStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= 500
}

func bodyContentType(headerValue string) string {
	if strings.TrimSpace(headerValue) == "" {
		return "application/octet-stream"
	}
	mediaType, _, err := mime.ParseMediaType(headerValue)
	if err != nil || mediaType == "" {
		return headerValue
	}
	return mediaType
}

type runnerErr struct {
	opErr model.OpError
}

func (e runnerErr) Error() string {
	return e.opErr.Message
}

func (e runnerErr) OpError() model.OpError {
	return e.opErr
}

func runnerError(code, message string, retryable bool, details json.RawMessage, now time.Time) error {
	return runnerErr{
		opErr: model.OpError{
			Code:       code,
			Message:    message,
			Retryable:  retryable,
			Details:    details,
			OccurredAt: now,
		},
	}
}

func mustMarshalRaw(value any) json.RawMessage {
	body, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return json.RawMessage(body)
}
