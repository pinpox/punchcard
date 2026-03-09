package main

import (
	"database/sql"
	"log"
	"net/http"
	"net/url"
	"time"
)

// handleLogin initiates the OIDC login flow
func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	state, err := a.auth.GenerateState()
	if err != nil {
		log.Printf("Error generating state: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Store state in session cookie for verification
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   300, // 5 minutes
	})

	authURL := a.auth.GetAuthURL(state)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// handleCallback handles the OIDC callback
func (a *App) handleCallback(w http.ResponseWriter, r *http.Request) {
	// Verify state parameter
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		log.Printf("No state cookie found: %v", err)
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" || state != stateCookie.Value {
		log.Printf("State mismatch: expected %s, got %s", stateCookie.Value, state)
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	// Check for error in callback
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		log.Printf("OIDC error: %s - %s", errMsg, r.URL.Query().Get("error_description"))
		http.Error(w, "Authentication failed", http.StatusBadRequest)
		return
	}

	// Get authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		log.Printf("No authorization code received")
		http.Error(w, "No authorization code received", http.StatusBadRequest)
		return
	}

	// Exchange code for tokens and get/create user
	user, err := a.auth.ExchangeCode(r.Context(), code)
	if err != nil {
		log.Printf("Error exchanging code: %v", err)
		http.Error(w, "Authentication failed", http.StatusInternalServerError)
		return
	}

	// Create session
	session, err := a.auth.CreateSession(user.ID)
	if err != nil {
		log.Printf("Error creating session: %v", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(session.ExpiresAt.Sub(session.CreatedAt).Seconds()),
	})

	// Redirect to original URL or home
	returnURL := "/"
	if returnCookie, err := r.Cookie("return_url"); err == nil {
		if decodedURL, err := url.QueryUnescape(returnCookie.Value); err == nil {
			returnURL = decodedURL
		}
		// Clear return URL cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "return_url",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   -1,
		})
	}

	log.Printf("User %s (%s) logged in successfully", user.Name, user.Email)
	http.Redirect(w, r, returnURL, http.StatusFound)
}

// handleLogout logs out the user
func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	sessionCookie, err := r.Cookie("session")
	if err == nil {
		// Delete session from database
		if err := a.auth.DeleteSession(sessionCookie.Value); err != nil {
			log.Printf("Error deleting session: %v", err)
		}
	}

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	// Redirect to login page
	http.Redirect(w, r, "/login", http.StatusFound)
}

// handleLoginPage serves the login page
func (a *App) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	// Check if user is already logged in
	if sessionCookie, err := r.Cookie("session"); err == nil {
		if user, err := a.auth.GetUserFromSession(sessionCookie.Value); err == nil && user != nil {
			// User is already logged in, redirect to home
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
	}

	// Render login page
	tmplData := struct {
		Title string
	}{
		Title: "Login - Punchcard",
	}

	if err := a.templates.ExecuteTemplate(w, "login.html", tmplData); err != nil {
		log.Printf("Error rendering login template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// handleUserProfile shows user profile information
func (a *App) handleUserProfile(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	if user == nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Get user statistics
	stats, err := a.getUserStats(user.ID)
	if err != nil {
		log.Printf("Error getting user stats: %v", err)
		stats = &UserStats{}
	}

	tmplData := struct {
		Title string
		User  *User
		Stats *UserStats
	}{
		Title: "Profile - Punchcard",
		User:  user,
		Stats: stats,
	}

	if err := a.templates.ExecuteTemplate(w, "profile.html", tmplData); err != nil {
		log.Printf("Error rendering profile template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// UserStats holds user statistics
type UserStats struct {
	TotalEntries      int     `json:"total_entries"`
	TotalHoursThisWeek float64 `json:"total_hours_this_week"`
	TotalHoursThisMonth float64 `json:"total_hours_this_month"`
	TotalHoursAllTime  float64 `json:"total_hours_all_time"`
}

// getUserStats calculates statistics for a user
func (a *App) getUserStats(userID int) (*UserStats, error) {
	var stats UserStats

	// Total entries
	err := a.db.QueryRow(`
		SELECT COUNT(*) FROM time_entries WHERE user_id = ?
	`, userID).Scan(&stats.TotalEntries)
	if err != nil {
		log.Printf("Error getting total entries: %v", err)
	}

	// Total hours this week
	now := todayStart()
	weekStart := now.AddDate(0, 0, -int(now.Weekday()))
	weekEnd := weekStart.AddDate(0, 0, 7)
	
	rows, err := a.db.Query(`
		SELECT start_time, end_time, is_running
		FROM time_entries 
		WHERE user_id = ? AND date >= ? AND date < ?
	`, userID, weekStart.Format("2006-01-02"), weekEnd.Format("2006-01-02"))
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var startTime, endTime sql.NullInt64
			var isRunning bool
			
			if err := rows.Scan(&startTime, &endTime, &isRunning); err == nil {
				entry := TimeEntry{
					IsRunning: isRunning,
				}
				if startTime.Valid {
					entry.StartTime = time.Unix(startTime.Int64, 0)
				}
				if endTime.Valid {
					entry.EndTime = time.Unix(endTime.Int64, 0)
				}
				
				stats.TotalHoursThisWeek += entry.Duration().Hours()
			}
		}
	}

	// Total hours this month
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthEnd := monthStart.AddDate(0, 1, 0)
	
	rows, err = a.db.Query(`
		SELECT start_time, end_time, is_running
		FROM time_entries 
		WHERE user_id = ? AND date >= ? AND date < ?
	`, userID, monthStart.Format("2006-01-02"), monthEnd.Format("2006-01-02"))
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var startTime, endTime sql.NullInt64
			var isRunning bool
			
			if err := rows.Scan(&startTime, &endTime, &isRunning); err == nil {
				entry := TimeEntry{
					IsRunning: isRunning,
				}
				if startTime.Valid {
					entry.StartTime = time.Unix(startTime.Int64, 0)
				}
				if endTime.Valid {
					entry.EndTime = time.Unix(endTime.Int64, 0)
				}
				
				stats.TotalHoursThisMonth += entry.Duration().Hours()
			}
		}
	}

	// Total hours all time
	rows, err = a.db.Query(`
		SELECT start_time, end_time, is_running
		FROM time_entries 
		WHERE user_id = ?
	`, userID)
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var startTime, endTime sql.NullInt64
			var isRunning bool
			
			if err := rows.Scan(&startTime, &endTime, &isRunning); err == nil {
				entry := TimeEntry{
					IsRunning: isRunning,
				}
				if startTime.Valid {
					entry.StartTime = time.Unix(startTime.Int64, 0)
				}
				if endTime.Valid {
					entry.EndTime = time.Unix(endTime.Int64, 0)
				}
				
				stats.TotalHoursAllTime += entry.Duration().Hours()
			}
		}
	}

	return &stats, nil
}