package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

type TimeEntry struct {
	ID          int       `json:"id"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	IsRunning   bool      `json:"is_running"`
}

func (e TimeEntry) Title() string {
	lines := strings.Split(e.Description, "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return e.Description
}

func (e TimeEntry) DescriptionLines() []string {
	lines := strings.Split(e.Description, "\n")
	if len(lines) <= 1 {
		return nil
	}
	
	var result []string
	for _, line := range lines[1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// Helper functions for date handling
func todayStart() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}



func (a *App) getTodaysEntries() []TimeEntry {
	return a.getEntriesForDate(todayStart())
}

func (a *App) getEntriesForDate(date time.Time) []TimeEntry {
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	dayEnd := dayStart.AddDate(0, 0, 1)
	
	query := `
		SELECT id, description, date, start_time, end_time, is_running 
		FROM time_entries 
		WHERE date >= ? AND date < ?
		ORDER BY start_time DESC
	`
	
	rows, err := a.db.Query(query, dayStart.Format("2006-01-02"), dayEnd.Format("2006-01-02"))
	if err != nil {
		log.Printf("Error querying entries: %v", err)
		return []TimeEntry{}
	}
	defer rows.Close()
	
	var entries []TimeEntry
	for rows.Next() {
		var entry TimeEntry
		var dateStr string
		var startTimeUnix, endTimeUnix sql.NullInt64
		
		err := rows.Scan(&entry.ID, &entry.Description, &dateStr, &startTimeUnix, &endTimeUnix, &entry.IsRunning)
		if err != nil {
			log.Printf("Error scanning entry: %v", err)
			continue
		}
		
		// Parse date
		if entry.Date, err = parseDate(dateStr); err != nil {
			log.Printf("Error parsing date: %v", err)
			continue
		}
		
		if startTimeUnix.Valid {
			entry.StartTime = time.Unix(startTimeUnix.Int64, 0)
		}
		
		if endTimeUnix.Valid {
			entry.EndTime = time.Unix(endTimeUnix.Int64, 0)
		}
		
		entries = append(entries, entry)
	}
	
	return entries
}

func (a *App) createEntry(description string, date time.Time) (TimeEntry, error) {
	query := `
		INSERT INTO time_entries (description, date, start_time, is_running)
		VALUES (?, ?, ?, ?)
	`
	
	now := time.Now()
	result, err := a.db.Exec(query, description, date.Format("2006-01-02"), now.Unix(), true)
	if err != nil {
		return TimeEntry{}, err
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return TimeEntry{}, err
	}
	
	return TimeEntry{
		ID:          int(id),
		Description: description,
		Date:        date,
		StartTime:   now,
		EndTime:     time.Time{},
		IsRunning:   true,
	}, nil
}

func (a *App) updateEntry(entry TimeEntry) error {
	query := `
		UPDATE time_entries 
		SET description = ?, start_time = ?, end_time = ?, is_running = ?
		WHERE id = ?
	`
	
	var startTime, endTime interface{}
	if !entry.StartTime.IsZero() {
		startTime = entry.StartTime.Unix()
	}
	if !entry.EndTime.IsZero() {
		endTime = entry.EndTime.Unix()
	}
	
	_, err := a.db.Exec(query, entry.Description, startTime, endTime, entry.IsRunning, entry.ID)
	return err
}

func (a *App) getEntryById(id int) (TimeEntry, error) {
	query := `
		SELECT id, description, date, start_time, end_time, is_running 
		FROM time_entries 
		WHERE id = ?
	`
	
	var entry TimeEntry
	var dateStr string
	var startTimeUnix, endTimeUnix sql.NullInt64
	
	err := a.db.QueryRow(query, id).Scan(&entry.ID, &entry.Description, &dateStr, &startTimeUnix, &endTimeUnix, &entry.IsRunning)
	if err != nil {
		return TimeEntry{}, err
	}
	
	// Parse date
	if entry.Date, err = parseDate(dateStr); err != nil {
		return TimeEntry{}, err
	}
	
	if startTimeUnix.Valid {
		entry.StartTime = time.Unix(startTimeUnix.Int64, 0)
	}
	
	if endTimeUnix.Valid {
		entry.EndTime = time.Unix(endTimeUnix.Int64, 0)
	}
	
	return entry, nil
}

func (a *App) deleteEntry(id int) error {
	query := `DELETE FROM time_entries WHERE id = ?`
	_, err := a.db.Exec(query, id)
	return err
}

func (a *App) stopRunningEntries(date time.Time) ([]TimeEntry, error) {
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	dayEnd := dayStart.AddDate(0, 0, 1)
	
	// Get running entries for this date
	query := `
		SELECT id, description, date, start_time, end_time, is_running 
		FROM time_entries 
		WHERE date >= ? AND date < ? AND is_running = true
	`
	
	rows, err := a.db.Query(query, dayStart.Format("2006-01-02"), dayEnd.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var stoppedEntries []TimeEntry
	for rows.Next() {
		var entry TimeEntry
		var dateStr string
		var startTimeUnix, endTimeUnix sql.NullInt64
		
		err := rows.Scan(&entry.ID, &entry.Description, &dateStr, &startTimeUnix, &endTimeUnix, &entry.IsRunning)
		if err != nil {
			log.Printf("Error scanning entry: %v", err)
			continue
		}
		
		// Parse date
		if entry.Date, err = parseDate(dateStr); err != nil {
			continue
		}
		if startTimeUnix.Valid {
			entry.StartTime = time.Unix(startTimeUnix.Int64, 0)
		}
		
		// Stop this entry
		entry.IsRunning = false
		entry.EndTime = time.Now()
		
		if err := a.updateEntry(entry); err != nil {
			log.Printf("Error stopping entry: %v", err)
			continue
		}
		
		stoppedEntries = append(stoppedEntries, entry)
	}
	
	return stoppedEntries, nil
}

func isSameDay(date1, date2 time.Time) bool {
	y1, m1, d1 := date1.Date()
	y2, m2, d2 := date2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

func parseDate(dateStr string) (time.Time, error) {
	// Try simple date format first
	if date, err := time.Parse("2006-01-02", dateStr); err == nil {
		return date, nil
	}
	
	// Try ISO format
	if date, err := time.Parse("2006-01-02T15:04:05Z", dateStr); err == nil {
		return date, nil
	}
	
	// Try RFC3339 format
	if date, err := time.Parse(time.RFC3339, dateStr); err == nil {
		return date, nil
	}
	
	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}



type DayStats struct {
	DayName    string
	Date       time.Time
	DateShort  string
	TotalHours float64
	BarHeight  int
	IsToday    bool
	IsViewDay  bool
}

type MonthDay struct {
	Date         time.Time
	DayNumber    int
	IsToday      bool
	IsInMonth    bool
	TotalHours   float64
	EntryCount   int
	FirstEntry   string // First entry title for display
	HasEntries   bool
}

type MonthStats struct {
	Year         int
	Month        time.Month
	MonthName    string
	Days         []MonthDay
	TotalHours   float64
	TotalEntries int
	PrevMonthURL string
	NextMonthURL string
}

func (a *App) getWeekStats(viewDate time.Time) []DayStats {
	// Find Monday of the week containing viewDate
	weekday := int(viewDate.Weekday())
	if weekday == 0 { // Sunday
		weekday = 7
	}
	monday := viewDate.AddDate(0, 0, -(weekday-1))
	
	var weekStats []DayStats
	var maxHours float64
	today := time.Now()
	
	// First pass: collect data and find max hours
	for i := 0; i < 7; i++ {
		day := monday.AddDate(0, 0, i)
		dayEntries := a.getEntriesForDate(day)
		
		// Calculate total duration for this day
		var totalDuration time.Duration
		for _, entry := range dayEntries {
			totalDuration += entry.Duration()
		}
		
		totalHours := totalDuration.Hours()
		if totalHours > maxHours {
			maxHours = totalHours
		}
		
		weekStats = append(weekStats, DayStats{
			DayName:    day.Format("Mon"),
			Date:       day,
			DateShort:  day.Format("02"),
			TotalHours: totalHours,
			IsToday:    isSameDay(day, today),
			IsViewDay:  isSameDay(day, viewDate),
		})
	}
	
	// Second pass: calculate bar heights
	const maxBarHeight = 60    // Maximum bar height in pixels
	const minBarHeight = 2     // Minimum bar height for visibility
	const minScaleThreshold = 1.0 // Hours threshold for proportional scaling
	
	for i := range weekStats {
		if weekStats[i].TotalHours > 0 {
			var barHeight int
			
			if maxHours < minScaleThreshold {
				// For small values, use linear scaling (pixels per hour)
				// 1 hour = 20 pixels when under threshold
				barHeight = int(weekStats[i].TotalHours * 20)
			} else {
				// Use proportional scaling for larger values
				proportion := weekStats[i].TotalHours / maxHours
				barHeight = int(proportion * maxBarHeight)
			}
			
			// Apply minimum height for visibility
			if barHeight < minBarHeight {
				barHeight = minBarHeight
			}
			// Cap at maximum height
			if barHeight > maxBarHeight {
				barHeight = maxBarHeight
			}
			
			weekStats[i].BarHeight = barHeight
		} else {
			weekStats[i].BarHeight = 0
		}
	}
	
	return weekStats
}

func (a *App) writeWeekChartUpdate(w http.ResponseWriter, viewDate time.Time) {
	weekStats := a.getWeekStats(viewDate)
	fmt.Fprintf(w, `<div hx-swap-oob="innerHTML:.chart-container">`)
	for _, stat := range weekStats {
		fmt.Fprintf(w, `<div class="day-bar`)
		if stat.IsToday {
			fmt.Fprintf(w, ` today`)
		}
		if stat.IsViewDay {
			fmt.Fprintf(w, ` selected`)
		}
		fmt.Fprintf(w, `" onclick="window.location.href='%s'" title="%s, %s: %.1fh">`, 
			stat.Date.Format("/2006-01-02"), 
			stat.DayName, 
			stat.Date.Format("2006-01-02"), 
			stat.TotalHours)
		fmt.Fprintf(w, `<div class="hours-label">`)
		if stat.TotalHours > 0.0 {
			fmt.Fprintf(w, `%.1fh`, stat.TotalHours)
		}
		fmt.Fprintf(w, `</div>`)
		fmt.Fprintf(w, `<div class="bar" style="height: %dpx;"></div>`, stat.BarHeight)
		fmt.Fprintf(w, `<div class="day-label">`)
		fmt.Fprintf(w, `<div class="day-name">%s</div>`, stat.DayName)
		fmt.Fprintf(w, `<div class="day-date">%s</div>`, stat.DateShort)
		fmt.Fprintf(w, `</div></div>`)
	}
	fmt.Fprintf(w, `</div>`)
}

func (a *App) getMonthStats(year int, month time.Month) MonthStats {
	// First day of the month
	firstDay := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	// Last day of the month
	lastDay := firstDay.AddDate(0, 1, 0).AddDate(0, 0, -1)
	
	// Start from Sunday of the week containing the first day
	startDate := firstDay
	for startDate.Weekday() != time.Sunday {
		startDate = startDate.AddDate(0, 0, -1)
	}
	
	// End on Saturday of the week containing the last day
	endDate := lastDay
	for endDate.Weekday() != time.Saturday {
		endDate = endDate.AddDate(0, 0, 1)
	}
	
	var monthDays []MonthDay
	var totalHours float64
	var totalEntries int
	today := time.Now()
	
	// Generate calendar grid (6 weeks max)
	current := startDate
	for current.Before(endDate.AddDate(0, 0, 1)) {
		dayEntries := a.getEntriesForDate(current)
		
		// Calculate total duration for this day
		var dayDuration time.Duration
		for _, entry := range dayEntries {
			dayDuration += entry.Duration()
		}
		
		dayHours := dayDuration.Hours()
		entryCount := len(dayEntries)
		
		var firstEntryTitle string
		if len(dayEntries) > 0 {
			firstEntryTitle = dayEntries[0].Title()
		}
		
		monthDay := MonthDay{
			Date:         current,
			DayNumber:    current.Day(),
			IsToday:      isSameDay(current, today),
			IsInMonth:    current.Month() == month,
			TotalHours:   dayHours,
			EntryCount:   entryCount,
			FirstEntry:   firstEntryTitle,
			HasEntries:   entryCount > 0,
		}
		
		monthDays = append(monthDays, monthDay)
		
		// Only count days in the current month for totals
		if current.Month() == month {
			totalHours += dayHours
			totalEntries += entryCount
		}
		
		current = current.AddDate(0, 0, 1)
	}
	
	// Previous and next month URLs
	prevMonth := firstDay.AddDate(0, -1, 0)
	nextMonth := firstDay.AddDate(0, 1, 0)
	
	return MonthStats{
		Year:         year,
		Month:        month,
		MonthName:    month.String(),
		Days:         monthDays,
		TotalHours:   totalHours,
		TotalEntries: totalEntries,
		PrevMonthURL: "/month/" + prevMonth.Format("2006-01"),
		NextMonthURL: "/month/" + nextMonth.Format("2006-01"),
	}
}

type App struct {
	db        *sql.DB
	templates *template.Template
}

func (e TimeEntry) Duration() time.Duration {
	if e.IsRunning {
		if e.StartTime.IsZero() {
			return 0
		}
		duration := time.Since(e.StartTime)
		if duration < 0 {
			return 0
		}
		return duration
	}
	if e.EndTime.IsZero() || e.StartTime.IsZero() {
		return 0
	}
	duration := e.EndTime.Sub(e.StartTime)
	if duration < 0 {
		return 0
	}
	return duration
}

func (e TimeEntry) DurationString() string {
	d := e.Duration()
	
	// Handle negative durations
	if d < 0 {
		return "00:00:00"
	}
	
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	
	// Ensure positive values
	if minutes < 0 {
		minutes = 0
	}
	if seconds < 0 {
		seconds = 0
	}
	
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func NewApp() *App {
	// Create template with custom functions
	funcMap := template.FuncMap{
		"sub": func(a, b int) int {
			return a - b
		},
		"div": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"workingDaysInMonth": func(year int, month time.Month) float64 {
			firstDay := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
			lastDay := firstDay.AddDate(0, 1, 0).AddDate(0, 0, -1)
			
			workingDays := 0
			for d := firstDay; !d.After(lastDay); d = d.AddDate(0, 0, 1) {
				if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
					workingDays++
				}
			}
			return float64(workingDays)
		},
	}
	
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.html"))
	
	// Initialize database
	db, err := initDatabase()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	
	return &App{
		db:        db,
		templates: tmpl,
	}
}

func initDatabase() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "punchcard.db")
	if err != nil {
		return nil, err
	}
	
	// Create tables
	if err := createTables(db); err != nil {
		return nil, err
	}
	
	return db, nil
}

func createTables(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS time_entries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		description TEXT NOT NULL,
		date DATE NOT NULL,
		start_time INTEGER,
		end_time INTEGER,
		is_running BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_date ON time_entries(date);
	CREATE INDEX IF NOT EXISTS idx_running ON time_entries(is_running);
	`
	
	_, err := db.Exec(schema)
	return err
}

func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Redirect root to today's date
	today := todayStart()
	todayURL := "/" + today.Format("2006-01-02")
	http.Redirect(w, r, todayURL, http.StatusTemporaryRedirect)
}

func (a *App) handleDateView(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dateStr := vars["date"]
	
	// Parse the date from URL
	viewDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, "Invalid date. Please use format YYYY-MM-DD with a valid date (e.g., 2025-03-11)", http.StatusBadRequest)
		return
	}
	
	// Get entries for this specific date
	dateEntries := a.getEntriesForDate(viewDate)
	
	// Calculate previous and next day URLs
	prevDay := viewDate.AddDate(0, 0, -1)
	nextDay := viewDate.AddDate(0, 0, 1)
	
	// Get week statistics
	weekStats := a.getWeekStats(viewDate)
	
	data := struct {
		Entries     []TimeEntry
		ViewDate    time.Time
		DayName     string
		DateISO     string
		PrevDayURL  string
		NextDayURL  string
		HasEntries  bool
		IsToday     bool
		WeekStats   []DayStats
		MonthURL    string
	}{
		Entries:     dateEntries,
		ViewDate:    viewDate,
		DayName:     viewDate.Format("Monday"),
		DateISO:     viewDate.Format("2006-01-02"),
		PrevDayURL:  "/" + prevDay.Format("2006-01-02"),
		NextDayURL:  "/" + nextDay.Format("2006-01-02"),
		HasEntries:  len(dateEntries) > 0,
		IsToday:     isSameDay(viewDate, time.Now()),
		WeekStats:   weekStats,
		MonthURL:    "/month/" + viewDate.Format("2006-01"),
	}
	
	if err := a.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (a *App) handleMonthView(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	monthStr := vars["month"]
	
	// Parse the month from URL (format: YYYY-MM)
	viewDate, err := time.Parse("2006-01", monthStr)
	if err != nil {
		http.Error(w, "Invalid month. Please use format YYYY-MM (e.g., 2026-03)", http.StatusBadRequest)
		return
	}
	
	// Get month statistics
	monthStats := a.getMonthStats(viewDate.Year(), viewDate.Month())
	
	data := struct {
		MonthStats  MonthStats
		ViewDate    time.Time
		TodayURL    string
	}{
		MonthStats:  monthStats,
		ViewDate:    viewDate,
		TodayURL:    "/" + time.Now().Format("2006-01-02"),
	}
	
	if err := a.templates.ExecuteTemplate(w, "month.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (a *App) handleAddEntry(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dateStr := vars["date"]
	
	// Parse the date from URL
	entryDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, "Invalid date. Please use format YYYY-MM-DD with a valid date", http.StatusBadRequest)
		return
	}
	
	description := r.FormValue("description")
	if description == "" {
		description = "Untitled Task"
	}

	// Stop any currently running timers for this date
	dayStart := time.Date(entryDate.Year(), entryDate.Month(), entryDate.Day(), 0, 0, 0, 0, entryDate.Location())
	stoppedEntries, err := a.stopRunningEntries(entryDate)
	if err != nil {
		log.Printf("Error stopping running entries: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create new entry and start it automatically
	entry, err := a.createEntry(description, dayStart)
	if err != nil {
		log.Printf("Error creating entry: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set content type and return response with out-of-band updates for stopped timers
	w.Header().Set("Content-Type", "text/html")
	
	// Return the new entry
	if err := a.templates.ExecuteTemplate(w, "time_entry.html", entry); err != nil {
		log.Printf("Error executing template: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Internal server error")
		return
	}
	
	// Add out-of-band updates for any stopped entries
	for _, stoppedEntry := range stoppedEntries {
		fmt.Fprintf(w, `<div hx-swap-oob="outerHTML:#entry-%d">`, stoppedEntry.ID)
		if err := a.templates.ExecuteTemplate(w, "time_entry.html", stoppedEntry); err != nil {
			log.Printf("Error executing template for stopped entry: %v", err)
			continue
		}
		fmt.Fprintf(w, `</div>`)
	}
	
	// Update the week chart with out-of-band swap
	a.writeWeekChartUpdate(w, entryDate)
}

func (a *App) handleStartStop(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dateStr := vars["date"]
	idStr := vars["id"]
	
	// Parse the date from URL
	viewDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, "Invalid date. Please use format YYYY-MM-DD with a valid date", http.StatusBadRequest)
		return
	}
	
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Find the target entry
	targetEntry, err := a.getEntryById(id)
	if err != nil {
		http.Error(w, "Entry not found", http.StatusNotFound)
		return
	}

	var stoppedEntries []TimeEntry

	if targetEntry.IsRunning {
		// Stop the target timer
		targetEntry.IsRunning = false
		targetEntry.EndTime = time.Now()
	} else {
		// Stop any other running timers from this date before starting this one
		stoppedEntries, err = a.stopRunningEntries(viewDate)
		if err != nil {
			log.Printf("Error stopping running entries: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Calculate accumulated time and restart timer
		var accumulatedTime time.Duration
		if !targetEntry.StartTime.IsZero() && !targetEntry.EndTime.IsZero() {
			// Timer has been stopped before, preserve accumulated time
			accumulatedTime = targetEntry.EndTime.Sub(targetEntry.StartTime)
		}

		// Start the timer - adjust StartTime to preserve accumulated duration
		targetEntry.IsRunning = true
		targetEntry.StartTime = time.Now().Add(-accumulatedTime)
		targetEntry.EndTime = time.Time{} // Reset end time
	}

	// Update the entry in database
	if err := a.updateEntry(targetEntry); err != nil {
		log.Printf("Error updating entry: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set content type and return updated row HTML
	w.Header().Set("Content-Type", "text/html")
	if err := a.templates.ExecuteTemplate(w, "time_entry.html", targetEntry); err != nil {
		log.Printf("Error executing template: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Internal server error")
		return
	}

	// Add out-of-band updates for any stopped entries
	for _, stoppedEntry := range stoppedEntries {
		fmt.Fprintf(w, `<div hx-swap-oob="outerHTML:#entry-%d">`, stoppedEntry.ID)
		if err := a.templates.ExecuteTemplate(w, "time_entry.html", stoppedEntry); err != nil {
			log.Printf("Error executing template for stopped entry: %v", err)
			continue
		}
		fmt.Fprintf(w, `</div>`)
	}
	
	// Update the week chart with out-of-band swap
	a.writeWeekChartUpdate(w, viewDate)
}

func (a *App) handleUpdateTimer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Find the entry and return its duration
	entry, err := a.getEntryById(id)
	if err != nil {
		http.Error(w, "Entry not found", http.StatusNotFound)
		return
	}

	fmt.Fprintf(w, entry.DurationString())
}

func (a *App) handleEditTime(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dateStr := vars["date"]
	idStr := vars["id"]
	
	// Parse the date from URL
	viewDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, "Invalid date. Please use format YYYY-MM-DD with a valid date", http.StatusBadRequest)
		return
	}
	
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if r.Method == "POST" {
		// Handle time update
		timeStr := r.FormValue("duration")
		duration, err := parseDuration(timeStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid time format: %v. Use single number for hours (e.g., '2') or H:MM format (e.g., '1:30')", err), http.StatusBadRequest)
			return
		}

		// Find and update the entry
		entry, err := a.getEntryById(id)
		if err != nil {
			http.Error(w, "Entry not found", http.StatusNotFound)
			return
		}

		if entry.IsRunning {
			http.Error(w, "Cannot edit time for running entry", http.StatusBadRequest)
			return
		}

		// Set manual duration by adjusting the times
		entry.StartTime = time.Now().Add(-duration)
		entry.EndTime = time.Now()

		// Update in database
		if err := a.updateEntry(entry); err != nil {
			log.Printf("Error updating entry: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		
		// Return updated entry
		w.Header().Set("Content-Type", "text/html")
		if err := a.templates.ExecuteTemplate(w, "time_entry.html", entry); err != nil {
			log.Printf("Error executing template: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "Internal server error")
			return
		}
		
		// Update the week chart with out-of-band swap
		a.writeWeekChartUpdate(w, viewDate)
	}
}

func (a *App) handleEditDescription(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	description := r.FormValue("description")
	if description == "" {
		http.Error(w, "Description cannot be empty", http.StatusBadRequest)
		return
	}

	// Find and update the entry
	entry, err := a.getEntryById(id)
	if err != nil {
		http.Error(w, "Entry not found", http.StatusNotFound)
		return
	}

	entry.Description = description

	// Update in database
	if err := a.updateEntry(entry); err != nil {
		log.Printf("Error updating entry: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	// Return updated entry
	w.Header().Set("Content-Type", "text/html")
	if err := a.templates.ExecuteTemplate(w, "time_entry.html", entry); err != nil {
		log.Printf("Error executing template: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Internal server error")
		return
	}
}

func (a *App) handleDeleteEntry(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dateStr := vars["date"]
	idStr := vars["id"]
	
	// Parse the date from URL
	viewDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, "Invalid date. Please use format YYYY-MM-DD with a valid date", http.StatusBadRequest)
		return
	}
	
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Delete the entry from database
	if err := a.deleteEntry(id); err != nil {
		http.Error(w, "Entry not found", http.StatusNotFound)
		return
	}
	
	// Set content type
	w.Header().Set("Content-Type", "text/html")
	
	// Check if we need to show empty state for this date
	dateEntries := a.getEntriesForDate(viewDate)
	if len(dateEntries) == 0 {
		// Return empty state HTML with out-of-band swap to replace entries container
		dayText := "this day"
		if isSameDay(viewDate, time.Now()) {
			dayText = "today"
		}
		fmt.Fprintf(w, `<div hx-swap-oob="innerHTML:#entries">
			<div class="empty-state" id="empty-state">
				<p>No time entries for %s yet. Add your first task above! 🚀</p>
			</div>
		</div>`, dayText)
	} else {
		// Just return empty response - HTMX will remove the element
		w.WriteHeader(http.StatusOK)
	}
	
	// Update the week chart with out-of-band swap
	a.writeWeekChartUpdate(w, viewDate)
}

// parseDuration parses simple time formats into time.Duration
// Supports: "2" (2 hours), "1:30" (1 hour 30 minutes)
func parseDuration(timeStr string) (time.Duration, error) {
	timeStr = strings.TrimSpace(timeStr)
	
	// Handle empty input
	if timeStr == "" {
		return 0, fmt.Errorf("empty time")
	}

	// Check for HH:MM format
	if strings.Contains(timeStr, ":") {
		parts := strings.Split(timeStr, ":")
		if len(parts) != 2 {
			return 0, fmt.Errorf("invalid format - use H:MM (e.g., 1:30)")
		}

		hours, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, fmt.Errorf("invalid hours")
		}

		minutes, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, fmt.Errorf("invalid minutes")
		}

		if minutes >= 60 || hours < 0 || minutes < 0 || hours > 23 {
			return 0, fmt.Errorf("hours must be 0-23, minutes must be 0-59")
		}

		return time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute, nil
	}
	
	// Single number = hours (can be decimal)
	hours, err := strconv.ParseFloat(timeStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid format - use decimal number for hours (e.g., 0.25, 1.5) or H:MM")
	}
	
	if hours < 0 || hours > 24 {
		return 0, fmt.Errorf("hours must be between 0 and 24")
	}
	
	return time.Duration(hours * float64(time.Hour)), nil
}

func main() {
	app := NewApp()

	r := mux.NewRouter()
	
	// Serve static files (CSS, JS)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	
	// Routes
	r.HandleFunc("/", app.handleIndex).Methods("GET")
	r.HandleFunc("/month/{month:[0-9]{4}-[0-9]{2}}", app.handleMonthView).Methods("GET")
	r.HandleFunc("/{date:[0-9]{4}-[0-9]{2}-[0-9]{2}}", app.handleDateView).Methods("GET")
	r.HandleFunc("/{date:[0-9]{4}-[0-9]{2}-[0-9]{2}}/add", app.handleAddEntry).Methods("POST")
	r.HandleFunc("/{date:[0-9]{4}-[0-9]{2}-[0-9]{2}}/start-stop/{id}", app.handleStartStop).Methods("POST")
	r.HandleFunc("/{date:[0-9]{4}-[0-9]{2}-[0-9]{2}}/update-timer/{id}", app.handleUpdateTimer).Methods("GET")
	r.HandleFunc("/{date:[0-9]{4}-[0-9]{2}-[0-9]{2}}/edit-time/{id}", app.handleEditTime).Methods("POST")
	r.HandleFunc("/{date:[0-9]{4}-[0-9]{2}-[0-9]{2}}/edit-description/{id}", app.handleEditDescription).Methods("POST")
	r.HandleFunc("/{date:[0-9]{4}-[0-9]{2}-[0-9]{2}}/delete/{id}", app.handleDeleteEntry).Methods("DELETE")

	fmt.Println("Server starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", r))
}