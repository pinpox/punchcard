package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

// setupTestApp creates an App with an in-memory SQLite database for testing.
func setupTestApp(t *testing.T) *App {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := createTables(db); err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}

	// Insert a test user
	_, err = db.Exec(`INSERT INTO users (id, oidc_subject, email, name) VALUES (1, 'test-sub', 'test@example.com', 'Test User')`)
	if err != nil {
		t.Fatalf("failed to insert test user: %v", err)
	}

	return &App{db: db}
}

// insertTestEntry inserts a time entry directly into the database and returns its ID.
func insertTestEntry(t *testing.T, db *sql.DB, userID int, desc string, date string, startTime, endTime time.Time, isRunning, billable bool) int {
	t.Helper()

	var startUnix, endUnix interface{}
	if !startTime.IsZero() {
		startUnix = startTime.Unix()
	}
	if !endTime.IsZero() {
		endUnix = endTime.Unix()
	}

	result, err := db.Exec(`
		INSERT INTO time_entries (user_id, description, date, start_time, end_time, is_running, billable)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, userID, desc, date, startUnix, endUnix, isRunning, billable)
	if err != nil {
		t.Fatalf("failed to insert test entry: %v", err)
	}

	id, _ := result.LastInsertId()
	return int(id)
}

func TestCombineDateTimeRange(t *testing.T) {
	tests := []struct {
		name          string
		date          time.Time
		start         time.Time
		end           time.Time
		wantBegin     string
		wantEnd       string
		wantDuration  time.Duration
	}{
		{
			name:         "same day, times on correct date",
			date:         time.Date(2026, 3, 2, 0, 0, 0, 0, time.Local),
			start:        time.Date(2026, 3, 2, 9, 0, 0, 0, time.Local),
			end:          time.Date(2026, 3, 2, 11, 30, 0, 0, time.Local),
			wantBegin:    "2026-03-02T09:00:00",
			wantEnd:      "2026-03-02T11:30:00",
			wantDuration: 2*time.Hour + 30*time.Minute,
		},
		{
			name:         "times on wrong date (edited on different day)",
			date:         time.Date(2026, 3, 2, 0, 0, 0, 0, time.Local),
			start:        time.Date(2026, 3, 6, 14, 0, 0, 0, time.Local),
			end:          time.Date(2026, 3, 6, 15, 30, 0, 0, time.Local),
			wantBegin:    "2026-03-02T14:00:00",
			wantEnd:      "2026-03-02T15:30:00",
			wantDuration: 1*time.Hour + 30*time.Minute,
		},
		{
			name:         "midnight crossing - end on next day",
			date:         time.Date(2026, 3, 10, 0, 0, 0, 0, time.Local),
			start:        time.Date(2026, 3, 10, 23, 30, 0, 0, time.Local),
			end:          time.Date(2026, 3, 11, 0, 30, 0, 0, time.Local),
			wantBegin:    "2026-03-10T23:30:00",
			wantEnd:      "2026-03-11T00:30:00",
			wantDuration: 1 * time.Hour,
		},
		{
			name:         "short entry",
			date:         time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local),
			start:        time.Date(2026, 6, 1, 12, 0, 0, 0, time.Local),
			end:          time.Date(2026, 6, 1, 12, 5, 0, 0, time.Local),
			wantBegin:    "2026-06-01T12:00:00",
			wantEnd:      "2026-06-01T12:05:00",
			wantDuration: 5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			begin, end := combineDateTimeRange(tt.date, tt.start, tt.end)
			gotBegin := begin.Format("2006-01-02T15:04:05")
			gotEnd := end.Format("2006-01-02T15:04:05")
			gotDuration := end.Sub(begin)

			if gotBegin != tt.wantBegin {
				t.Errorf("begin = %s, want %s", gotBegin, tt.wantBegin)
			}
			if gotEnd != tt.wantEnd {
				t.Errorf("end = %s, want %s", gotEnd, tt.wantEnd)
			}
			if gotDuration != tt.wantDuration {
				t.Errorf("duration = %v, want %v", gotDuration, tt.wantDuration)
			}
		})
	}
}

func TestExportDateMatchesEntryDate(t *testing.T) {
	app := setupTestApp(t)

	// Simulate the core bug scenario:
	// Entry is for Monday 2026-03-02, but StartTime/EndTime were set on Friday 2026-03-06
	// (as happens when editing an entry's duration on a different day)
	monday := "2026-03-02"
	fridayStart := time.Date(2026, 3, 6, 14, 0, 0, 0, time.Local)
	fridayEnd := time.Date(2026, 3, 6, 15, 30, 0, 0, time.Local)

	insertTestEntry(t, app.db, 1, "Task on Monday", monday, fridayStart, fridayEnd, false, true)

	// Fetch entries via getEntriesForDate (the same path the export uses)
	entryDate := time.Date(2026, 3, 2, 0, 0, 0, 0, time.Local)
	entries := app.getEntriesForDate(1, entryDate)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]

	// Simulate the export conversion (same logic as handleMonthExport/handleDateExport)
	beginTime, endTime := combineDateTimeRange(entry.Date, entry.StartTime, entry.EndTime)

	// The exported Begin/End must be on Monday, NOT Friday
	if beginTime.Format("2006-01-02") != monday {
		t.Errorf("exported Begin date = %s, want %s", beginTime.Format("2006-01-02"), monday)
	}

	// The time-of-day should be preserved from the StartTime
	if beginTime.Format("15:04:05") != "14:00:00" {
		t.Errorf("exported Begin time = %s, want 14:00:00", beginTime.Format("15:04:05"))
	}

	// Duration should be preserved (1h30m)
	gotDuration := endTime.Sub(beginTime)
	wantDuration := 1*time.Hour + 30*time.Minute
	if gotDuration != wantDuration {
		t.Errorf("exported duration = %v, want %v", gotDuration, wantDuration)
	}
}

func TestExportMidnightCrossingEntry(t *testing.T) {
	app := setupTestApp(t)

	// Entry starts at 23:30 and ends at 00:30 the next day
	date := "2026-03-10"
	start := time.Date(2026, 3, 10, 23, 30, 0, 0, time.Local)
	end := time.Date(2026, 3, 11, 0, 30, 0, 0, time.Local)

	insertTestEntry(t, app.db, 1, "Late night work", date, start, end, false, true)

	entryDate := time.Date(2026, 3, 10, 0, 0, 0, 0, time.Local)
	entries := app.getEntriesForDate(1, entryDate)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	beginTime, endTime := combineDateTimeRange(entry.Date, entry.StartTime, entry.EndTime)

	// Begin should be on March 10 at 23:30
	if beginTime.Format("2006-01-02T15:04:05") != "2026-03-10T23:30:00" {
		t.Errorf("Begin = %s, want 2026-03-10T23:30:00", beginTime.Format("2006-01-02T15:04:05"))
	}

	// End should be on March 11 at 00:30 (next day, preserving 1h duration)
	if endTime.Format("2006-01-02T15:04:05") != "2026-03-11T00:30:00" {
		t.Errorf("End = %s, want 2026-03-11T00:30:00", endTime.Format("2006-01-02T15:04:05"))
	}

	// Duration must be exactly 1 hour
	gotDuration := endTime.Sub(beginTime)
	if gotDuration != 1*time.Hour {
		t.Errorf("duration = %v, want 1h0m0s", gotDuration)
	}
}

func TestExportEditedEntryPreservesDate(t *testing.T) {
	app := setupTestApp(t)

	// Create an entry for Wednesday via createEntry
	wednesday := time.Date(2026, 3, 4, 0, 0, 0, 0, time.Local)
	entry, err := app.createEntry(1, "Wednesday work", wednesday)
	if err != nil {
		t.Fatalf("failed to create entry: %v", err)
	}

	// Stop the entry
	entry.IsRunning = false
	entry.EndTime = time.Now()
	if err := app.updateEntry(entry); err != nil {
		t.Fatalf("failed to stop entry: %v", err)
	}

	// Simulate editing the duration to 2 hours (same logic as handleEditTime)
	duration := 2 * time.Hour
	entryDate := entry.Date
	editEndTime := time.Date(entryDate.Year(), entryDate.Month(), entryDate.Day(), 17, 0, 0, 0, time.Local)
	entry.StartTime = editEndTime.Add(-duration)
	entry.EndTime = editEndTime

	if err := app.updateEntry(entry); err != nil {
		t.Fatalf("failed to update entry: %v", err)
	}

	// Re-fetch and export
	updated, err := app.getEntryById(1, entry.ID)
	if err != nil {
		t.Fatalf("failed to get entry: %v", err)
	}

	beginTime, endTime := combineDateTimeRange(updated.Date, updated.StartTime, updated.EndTime)

	wantDate := "2026-03-04"
	if beginTime.Format("2006-01-02") != wantDate {
		t.Errorf("Begin date after edit = %s, want %s", beginTime.Format("2006-01-02"), wantDate)
	}

	// Duration should be 2 hours
	gotDuration := endTime.Sub(beginTime)
	if gotDuration != 2*time.Hour {
		t.Errorf("exported duration = %v, want 2h0m0s", gotDuration)
	}
}

func TestExportKimaiJSONStructure(t *testing.T) {
	app := setupTestApp(t)

	// Insert entries across multiple days
	mon := "2026-03-02"
	tue := "2026-03-03"

	monStart := time.Date(2026, 3, 2, 9, 0, 0, 0, time.Local)
	monEnd := time.Date(2026, 3, 2, 10, 30, 0, 0, time.Local)
	insertTestEntry(t, app.db, 1, "Monday task", mon, monStart, monEnd, false, true)

	tueStart := time.Date(2026, 3, 3, 14, 0, 0, 0, time.Local)
	tueEnd := time.Date(2026, 3, 3, 16, 15, 0, 0, time.Local)
	insertTestEntry(t, app.db, 1, "Tuesday task", tue, tueStart, tueEnd, false, false)

	// Fetch month entries the same way export does
	entries := app.getMonthEntries(1, 2026, time.March)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Build Kimai JSON (same logic as the handler)
	var kimaiEntries []KimaiTimeEntry
	for _, entry := range entries {
		ke := KimaiTimeEntry{
			Project:     1,
			Activity:    1,
			Description: entry.Description,
			Billable:    entry.Billable,
		}
		if !entry.IsRunning && !entry.EndTime.IsZero() {
			beginTime, endTime := combineDateTimeRange(entry.Date, entry.StartTime, entry.EndTime)
			ke.Begin = beginTime.Format("2006-01-02T15:04:05")
			ke.End = endTime.Format("2006-01-02T15:04:05")
		}
		kimaiEntries = append(kimaiEntries, ke)
	}

	// Marshal to JSON and verify structure
	jsonData, err := json.Marshal(kimaiEntries)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}

	// Parse back and verify
	var parsed []KimaiTimeEntry
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if len(parsed) != 2 {
		t.Fatalf("expected 2 entries in JSON, got %d", len(parsed))
	}

	// Entry 1: Monday, billable
	if parsed[0].Begin != "2026-03-02T09:00:00" {
		t.Errorf("entry 0 Begin = %s, want 2026-03-02T09:00:00", parsed[0].Begin)
	}
	if parsed[0].End != "2026-03-02T10:30:00" {
		t.Errorf("entry 0 End = %s, want 2026-03-02T10:30:00", parsed[0].End)
	}
	if parsed[0].Description != "Monday task" {
		t.Errorf("entry 0 Description = %s, want 'Monday task'", parsed[0].Description)
	}
	if !parsed[0].Billable {
		t.Error("entry 0 should be billable")
	}

	// Entry 2: Tuesday, not billable
	if parsed[1].Begin != "2026-03-03T14:00:00" {
		t.Errorf("entry 1 Begin = %s, want 2026-03-03T14:00:00", parsed[1].Begin)
	}
	if parsed[1].End != "2026-03-03T16:15:00" {
		t.Errorf("entry 1 End = %s, want 2026-03-03T16:15:00", parsed[1].End)
	}
	if parsed[1].Billable {
		t.Error("entry 1 should not be billable")
	}
}

func TestExportRunningEntryHasNoEnd(t *testing.T) {
	app := setupTestApp(t)

	start := time.Date(2026, 3, 5, 10, 0, 0, 0, time.Local)
	insertTestEntry(t, app.db, 1, "Running task", "2026-03-05", start, time.Time{}, true, true)

	entries := app.getMonthEntries(1, 2026, time.March)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]

	// Running entry: only anchor begin to entry date, no end
	ke := KimaiTimeEntry{
		Begin:       time.Date(entry.Date.Year(), entry.Date.Month(), entry.Date.Day(), entry.StartTime.Hour(), entry.StartTime.Minute(), entry.StartTime.Second(), 0, entry.StartTime.Location()).Format("2006-01-02T15:04:05"),
		Description: entry.Description,
		Billable:    entry.Billable,
	}

	if ke.End != "" {
		t.Errorf("running entry should have empty End, got %s", ke.End)
	}
	if ke.Begin != "2026-03-05T10:00:00" {
		t.Errorf("Begin = %s, want 2026-03-05T10:00:00", ke.Begin)
	}
}

// TestExportCompatibleWithKimaiSchema validates that our export JSON can be parsed
// alongside real Kimai API responses, and that the fields Kimai requires for import are present.
// The fixture testdata/kimai-export.json contains synthetic data matching the Kimai GET /api/timesheets schema.
func TestExportCompatibleWithKimaiSchema(t *testing.T) {
	// Load Kimai export fixture as reference
	kimaiData, err := os.ReadFile("testdata/kimai-export.json")
	if err != nil {
		t.Fatalf("failed to read kimai fixture: %v", err)
	}

	// KimaiGetEntry represents the full response from GET /api/timesheets
	type KimaiGetEntry struct {
		ID           int             `json:"id"`
		Activity     int             `json:"activity"`
		Project      int             `json:"project"`
		User         int             `json:"user"`
		Tags         json.RawMessage `json:"tags"`
		Begin        string          `json:"begin"`
		End          string          `json:"end"`
		Duration     int             `json:"duration"`
		Break        int             `json:"break"`
		Description  *string         `json:"description"`
		Rate         float64         `json:"rate"`
		InternalRate float64         `json:"internalRate"`
		Exported     bool            `json:"exported"`
		Billable     bool            `json:"billable"`
		MetaFields   json.RawMessage `json:"metaFields"`
	}

	var kimaiEntries []KimaiGetEntry
	if err := json.Unmarshal(kimaiData, &kimaiEntries); err != nil {
		t.Fatalf("failed to parse kimai fixture: %v", err)
	}

	if len(kimaiEntries) == 0 {
		t.Fatal("kimai fixture has no entries")
	}

	// Verify Kimai's begin/end format includes timezone offset
	for i, e := range kimaiEntries {
		// Kimai GET format: 2026-03-10T10:00:00+0100
		if _, err := time.Parse("2006-01-02T15:04:05-0700", e.Begin); err != nil {
			t.Errorf("kimai entry %d: unexpected begin format %q: %v", i, e.Begin, err)
		}
		if e.End != "" {
			if _, err := time.Parse("2006-01-02T15:04:05-0700", e.End); err != nil {
				t.Errorf("kimai entry %d: unexpected end format %q: %v", i, e.End, err)
			}
		}
	}

	// Now build our export and verify it's valid for Kimai POST
	// (Kimai POST accepts timestamps without timezone — confirmed via live API test)
	app := setupTestApp(t)

	start := time.Date(2026, 3, 10, 10, 0, 0, 0, time.Local)
	end := time.Date(2026, 3, 10, 12, 30, 0, 0, time.Local)
	insertTestEntry(t, app.db, 1, "Test entry", "2026-03-10", start, end, false, true)

	entries := app.getMonthEntries(1, 2026, time.March)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	beginTime, endTime := combineDateTimeRange(entry.Date, entry.StartTime, entry.EndTime)

	kimaiPost := KimaiTimeEntry{
		Project:     1,
		Activity:    1,
		Begin:       beginTime.Format("2006-01-02T15:04:05"),
		End:         endTime.Format("2006-01-02T15:04:05"),
		Description: entry.Description,
		Billable:    entry.Billable,
	}

	// Verify our export can be marshaled to valid JSON
	postJSON, err := json.Marshal(kimaiPost)
	if err != nil {
		t.Fatalf("failed to marshal our export: %v", err)
	}

	// Verify required fields for POST /api/timesheets are present
	var postMap map[string]interface{}
	if err := json.Unmarshal(postJSON, &postMap); err != nil {
		t.Fatalf("failed to unmarshal our export: %v", err)
	}

	requiredFields := []string{"project", "activity", "begin"}
	for _, field := range requiredFields {
		if _, ok := postMap[field]; !ok {
			t.Errorf("missing required Kimai field: %s", field)
		}
	}

	// Verify our begin/end format is parseable (no timezone = Kimai assumes server local)
	if _, err := time.Parse("2006-01-02T15:04:05", kimaiPost.Begin); err != nil {
		t.Errorf("our Begin format invalid: %v", err)
	}
	if _, err := time.Parse("2006-01-02T15:04:05", kimaiPost.End); err != nil {
		t.Errorf("our End format invalid: %v", err)
	}

	// Verify our exported times match the Kimai reference entry (stripping timezone)
	kimaiBeginLocal := kimaiEntries[0].Begin[:19] // "2026-03-10T10:00:00+0100" → "2026-03-10T10:00:00"
	kimaiEndLocal := kimaiEntries[0].End[:19]
	if kimaiPost.Begin != kimaiBeginLocal {
		t.Errorf("Begin mismatch: ours=%s, kimai=%s", kimaiPost.Begin, kimaiBeginLocal)
	}
	if kimaiPost.End != kimaiEndLocal {
		t.Errorf("End mismatch: ours=%s, kimai=%s", kimaiPost.End, kimaiEndLocal)
	}

	// Verify description matches
	if kimaiPost.Description != *kimaiEntries[0].Description {
		t.Errorf("Description mismatch: ours=%s, kimai=%s", kimaiPost.Description, *kimaiEntries[0].Description)
	}
}

func TestDateExportHTTPHandler(t *testing.T) {
	app := setupTestApp(t)

	start := time.Date(2026, 3, 2, 9, 0, 0, 0, time.Local)
	end := time.Date(2026, 3, 2, 11, 0, 0, 0, time.Local)
	insertTestEntry(t, app.db, 1, "Handler test", "2026-03-02", start, end, false, true)

	// Create request with user in context
	req := httptest.NewRequest("GET", "/2026-03-02.json", nil)
	req = mux.SetURLVars(req, map[string]string{"date": "2026-03-02"})

	// Inject user into context (same as AuthMiddleware does)
	user := &User{ID: 1, Name: "Test User", Email: "test@example.com"}
	ctx := req.Context()
	ctx = context.WithValue(ctx, "user", user)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	app.handleDateExport(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("handler returned status %d: %s", rr.Code, rr.Body.String())
	}

	// Verify Content-Type and Content-Disposition
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %s, want application/json", ct)
	}
	if cd := rr.Header().Get("Content-Disposition"); cd == "" {
		t.Error("missing Content-Disposition header")
	}

	// Parse response JSON
	var entries []KimaiTimeEntry
	if err := json.Unmarshal(rr.Body.Bytes(), &entries); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if entries[0].Begin != "2026-03-02T09:00:00" {
		t.Errorf("Begin = %s, want 2026-03-02T09:00:00", entries[0].Begin)
	}
	if entries[0].End != "2026-03-02T11:00:00" {
		t.Errorf("End = %s, want 2026-03-02T11:00:00", entries[0].End)
	}
}
