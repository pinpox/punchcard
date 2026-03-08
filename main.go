package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
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
	
	var dayEntries []TimeEntry
	for _, entry := range a.entries {
		if (entry.Date.Equal(dayStart) || entry.Date.After(dayStart)) && entry.Date.Before(dayEnd) {
			dayEntries = append(dayEntries, entry)
		}
	}
	return dayEntries
}

func isSameDay(date1, date2 time.Time) bool {
	y1, m1, d1 := date1.Date()
	y2, m2, d2 := date2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
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
	const maxBarHeight = 60 // Maximum bar height in pixels
	const minBarHeight = 2  // Minimum bar height for visibility
	
	for i := range weekStats {
		if weekStats[i].TotalHours > 0 && maxHours > 0 {
			// Scale bar height proportionally
			proportion := weekStats[i].TotalHours / maxHours
			weekStats[i].BarHeight = int(proportion*maxBarHeight)
			if weekStats[i].BarHeight < minBarHeight {
				weekStats[i].BarHeight = minBarHeight
			}
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

type App struct {
	entries   []TimeEntry
	nextID    int
	templates *template.Template
}

func (e TimeEntry) Duration() time.Duration {
	if e.IsRunning {
		return time.Since(e.StartTime)
	}
	if e.EndTime.IsZero() || e.StartTime.IsZero() {
		return 0
	}
	return e.EndTime.Sub(e.StartTime)
}

func (e TimeEntry) DurationString() string {
	d := e.Duration()
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func NewApp() *App {
	tmpl := template.Must(template.ParseGlob("templates/*.html"))
	return &App{
		entries:   make([]TimeEntry, 0),
		nextID:    1,
		templates: tmpl,
	}
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
	}
	
	if err := a.templates.ExecuteTemplate(w, "index.html", data); err != nil {
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
	var stoppedEntries []TimeEntry
	dayStart := time.Date(entryDate.Year(), entryDate.Month(), entryDate.Day(), 0, 0, 0, 0, entryDate.Location())
	dayEnd := dayStart.AddDate(0, 0, 1)
	
	for i := range a.entries {
		if a.entries[i].IsRunning && 
		   (a.entries[i].Date.Equal(dayStart) || a.entries[i].Date.After(dayStart)) && 
		   a.entries[i].Date.Before(dayEnd) {
			a.entries[i].IsRunning = false
			a.entries[i].EndTime = time.Now()
			stoppedEntries = append(stoppedEntries, a.entries[i])
		}
	}

	// Create new entry and start it automatically
	entry := TimeEntry{
		ID:          a.nextID,
		Description: description,
		Date:        dayStart, // Set to the viewed date
		StartTime:   time.Now(), // Start immediately
		EndTime:     time.Time{}, // Zero end time
		IsRunning:   true,        // Start running
	}
	a.nextID++
	a.entries = append(a.entries, entry)

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
	weekStats := a.getWeekStats(entryDate)
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

	var targetEntry *TimeEntry
	var stoppedEntries []TimeEntry

	// Find the target entry
	for i := range a.entries {
		if a.entries[i].ID == id {
			targetEntry = &a.entries[i]
			break
		}
	}

	if targetEntry == nil {
		http.Error(w, "Entry not found", http.StatusNotFound)
		return
	}

	if targetEntry.IsRunning {
		// Stop the target timer
		targetEntry.IsRunning = false
		targetEntry.EndTime = time.Now()
	} else {
		// Stop any other running timers from this date before starting this one
		dayStart := time.Date(viewDate.Year(), viewDate.Month(), viewDate.Day(), 0, 0, 0, 0, viewDate.Location())
		dayEnd := dayStart.AddDate(0, 0, 1)
		
		for i := range a.entries {
			if a.entries[i].IsRunning && a.entries[i].ID != id &&
			   (a.entries[i].Date.Equal(dayStart) || a.entries[i].Date.After(dayStart)) && 
			   a.entries[i].Date.Before(dayEnd) {
				a.entries[i].IsRunning = false
				a.entries[i].EndTime = time.Now()
				stoppedEntries = append(stoppedEntries, a.entries[i])
			}
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

	// Set content type and return updated row HTML
	w.Header().Set("Content-Type", "text/html")
	if err := a.templates.ExecuteTemplate(w, "time_entry.html", *targetEntry); err != nil {
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

func (a *App) handleUpdateTimer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Find the running entry and return just the duration
	for _, entry := range a.entries {
		if entry.ID == id && entry.IsRunning {
			fmt.Fprintf(w, entry.DurationString())
			return
		}
	}

	// If not running, return the static duration
	for _, entry := range a.entries {
		if entry.ID == id {
			fmt.Fprintf(w, entry.DurationString())
			return
		}
	}

	http.Error(w, "Entry not found", http.StatusNotFound)
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
		for i := range a.entries {
			if a.entries[i].ID == id && !a.entries[i].IsRunning {
				// Set manual duration by adjusting the times
				a.entries[i].StartTime = time.Now().Add(-duration)
				a.entries[i].EndTime = time.Now()
				
				// Return updated entry
				w.Header().Set("Content-Type", "text/html")
				if err := a.templates.ExecuteTemplate(w, "time_entry.html", a.entries[i]); err != nil {
					log.Printf("Error executing template: %v", err)
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprint(w, "Internal server error")
					return
				}
				
				// Update the week chart with out-of-band swap
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
				
				return
			}
		}
		http.Error(w, "Entry not found or is running", http.StatusNotFound)
	}
}

func (a *App) handleEditDescription(w http.ResponseWriter, r *http.Request) {
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

	description := r.FormValue("description")
	if description == "" {
		http.Error(w, "Description cannot be empty", http.StatusBadRequest)
		return
	}

	// Find and update the entry
	for i := range a.entries {
		if a.entries[i].ID == id {
			a.entries[i].Description = description
			
			// Return updated entry
			w.Header().Set("Content-Type", "text/html")
			if err := a.templates.ExecuteTemplate(w, "time_entry.html", a.entries[i]); err != nil {
				log.Printf("Error executing template: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, "Internal server error")
				return
			}
			return
		}
	}

	http.Error(w, "Entry not found", http.StatusNotFound)
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

	// Find and remove the entry
	for i, entry := range a.entries {
		if entry.ID == id {
			// Remove entry from slice
			a.entries = append(a.entries[:i], a.entries[i+1:]...)
			
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
			return
		}
	}

	http.Error(w, "Entry not found", http.StatusNotFound)
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
	
	// Single number = hours
	hours, err := strconv.Atoi(timeStr)
	if err != nil {
		return 0, fmt.Errorf("invalid format - use single number for hours or H:MM")
	}
	
	if hours < 0 || hours > 24 {
		return 0, fmt.Errorf("hours must be between 0 and 24")
	}
	
	return time.Duration(hours) * time.Hour, nil
}

func main() {
	app := NewApp()

	r := mux.NewRouter()
	
	// Serve static files (CSS, JS)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	
	// Routes
	r.HandleFunc("/", app.handleIndex).Methods("GET")
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