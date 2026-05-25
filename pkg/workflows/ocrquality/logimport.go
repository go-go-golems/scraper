package ocrquality

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type logRow struct {
	LineNo         int
	Level          string
	Event          string
	WorkflowID     string
	OpID           string
	Site           string
	Queue          string
	Attempt        any
	WorkflowStatus string
	Message        string
	ErrorCode      string
	Retryable      string
	Time           string
	Raw            string
	Parsed         bool
}

func ImportLogFile(input LogImportInput) (LogImportResult, error) {
	if strings.TrimSpace(input.LogPath) == "" {
		return LogImportResult{}, fmt.Errorf("log_path is required")
	}
	file, err := os.Open(input.LogPath)
	if err != nil {
		return LogImportResult{}, err
	}
	defer func() { _ = file.Close() }()

	var db *sql.DB
	if strings.TrimSpace(input.SQLitePath) != "" {
		if err := os.MkdirAll(filepath.Dir(input.SQLitePath), 0o755); err != nil {
			return LogImportResult{}, err
		}
		db, err = sql.Open("sqlite3", input.SQLitePath)
		if err != nil {
			return LogImportResult{}, err
		}
		defer func() { _ = db.Close() }()
		if err := initLogDB(db); err != nil {
			return LogImportResult{}, err
		}
	}

	result := LogImportResult{LogPath: input.LogPath, SQLitePath: input.SQLitePath, LevelCounts: map[string]int{}}
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 16*1024*1024)
	for scanner.Scan() {
		result.TotalLines++
		row := parseLogLine(result.TotalLines, scanner.Text())
		if row.Parsed {
			result.ParsedLines++
		}
		levelKey := row.Level
		if levelKey == "" {
			levelKey = "none"
		}
		result.LevelCounts[levelKey]++
		if row.Level == "trace" {
			result.TraceLines++
		}
		if row.Level == "warn" || row.Level == "error" || row.Level == "fatal" || row.Level == "panic" {
			result.WarningErrorLines++
		}
		if row.Event != "" && row.Level != "trace" {
			result.NonTraceEventLines++
		}
		if db != nil {
			if err := insertLogRow(db, row); err != nil {
				return LogImportResult{}, err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return LogImportResult{}, err
	}
	result.ReportMarkdown = RenderLogImportReport(result)
	return result, nil
}

func parseLogLine(lineNo int, raw string) logRow {
	row := logRow{LineNo: lineNo, Raw: raw}
	var obj map[string]any
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		row.Message = raw
		return row
	}
	row.Parsed = true
	row.Level = stringField(obj, "level")
	row.Event = stringField(obj, "event")
	row.WorkflowID = stringField(obj, "workflow_id")
	row.OpID = stringField(obj, "op_id")
	row.Site = stringField(obj, "site")
	row.Queue = stringField(obj, "queue")
	row.Attempt = obj["attempt"]
	row.WorkflowStatus = stringField(obj, "workflow_status")
	row.Message = firstStringField(obj, "message", "msg")
	row.ErrorCode = stringField(obj, "error_code")
	row.Retryable = fmt.Sprint(obj["retryable"])
	if row.Retryable == "<nil>" {
		row.Retryable = ""
	}
	row.Time = stringField(obj, "time")
	return row
}

func stringField(obj map[string]any, key string) string {
	value, _ := obj[key].(string)
	return value
}

func firstStringField(obj map[string]any, keys ...string) string {
	for _, key := range keys {
		if v := stringField(obj, key); v != "" {
			return v
		}
	}
	return ""
}

func initLogDB(db *sql.DB) error {
	stmts := []string{
		`drop table if exists log_events`,
		`create table log_events (line_no integer primary key, level text, event text, workflow_id text, op_id text, site text, queue text, attempt text, workflow_status text, message text, error_code text, retryable text, time text, raw text not null, parsed integer not null)`,
		`create index idx_log_level on log_events(level)`,
		`create index idx_log_event on log_events(event)`,
		`create index idx_log_op on log_events(op_id)`,
		`create index idx_log_workflow on log_events(workflow_id)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func insertLogRow(db *sql.DB, row logRow) error {
	parsed := 0
	if row.Parsed {
		parsed = 1
	}
	_, err := db.Exec(`insert into log_events values (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, row.LineNo, row.Level, row.Event, row.WorkflowID, row.OpID, row.Site, row.Queue, fmt.Sprint(row.Attempt), row.WorkflowStatus, row.Message, row.ErrorCode, row.Retryable, row.Time, row.Raw, parsed)
	return err
}

func RenderLogImportReport(result LogImportResult) string {
	var b strings.Builder
	b.WriteString("---\ndocType: reference\nstatus: active\nintent: short-term\ntopics:\n  - ocr\n  - experiments\ncreated: 2026-05-24\nupdated: 2026-05-24\n---\n\n")
	b.WriteString("# OCR Log Import Summary\n\n")
	fmt.Fprintf(&b, "- Log path: `%s`\n", result.LogPath)
	if result.SQLitePath != "" {
		fmt.Fprintf(&b, "- SQLite path: `%s`\n", result.SQLitePath)
	}
	fmt.Fprintf(&b, "- Total lines: %d\n", result.TotalLines)
	fmt.Fprintf(&b, "- Parsed JSON lines: %d\n", result.ParsedLines)
	fmt.Fprintf(&b, "- Trace lines: %d\n", result.TraceLines)
	fmt.Fprintf(&b, "- Non-trace workflow events: %d\n", result.NonTraceEventLines)
	fmt.Fprintf(&b, "- Warning/error lines: %d\n\n", result.WarningErrorLines)
	b.WriteString("## Level counts\n\n")
	for _, level := range sortedKeys(result.LevelCounts) {
		fmt.Fprintf(&b, "- `%s`: %d\n", level, result.LevelCounts[level])
	}
	return b.String()
}
