# Punchcard - Daily Time Tracker

A simple daily time tracking webapp built with Go and HTMX.

<img width="1151" height="745" alt="image" src="https://github.com/user-attachments/assets/d8b84a54-ff2f-4e69-9541-ea02547f9520" />

<img width="1276" height="1033" alt="image" src="https://github.com/user-attachments/assets/25372777-0827-416f-8557-b75db95762cf" />



## Features

- ✅ Daily time tracking with date display (shows current day)
- ✅ Weekly bar chart showing hours worked each day
- ✅ Previous/Next day navigation with keyboard shortcuts (← →)
- ✅ Calendar date picker (click on date to open)
- ✅ Quick "Today" button when viewing other dates
- ✅ Date validation with leap year support (automatically handles invalid dates)
- ✅ Add new time tracking tasks (auto-start timers)
- ✅ Start/Stop timers for each task
- ✅ Quick inline time editing (click timer, type "2" or "1:30", Enter or click away to save)
- ✅ Auto-expanding editable titles and descriptions (click away to save, no scrollbars)
- ✅ Delete entries with confirmation
- ✅ Single timer mode (only one running at a time per day)
- ✅ Real-time timer updates (every second)
- ✅ Clean, minimal UI focused on the essentials
- ✅ Keyboard shortcuts for navigation (arrow keys)
- ✅ No page refreshes (HTMX powered)

## Quick Start

1. **Run the application:**
   ```bash
   go run main.go
   ```

2. **Open your browser:**
   Navigate to `http://localhost:8080` (redirects to today)
   Or go directly to a specific day: `http://localhost:8080/2025-03-11`
   
   **Note:** Invalid dates like `2023-02-29` (non-leap year) will show an error

3. **Start tracking time:**
   - Add a new task using the input field (automatically starts timer)
   - Click "Start" to begin timing existing tasks
   - Click "Stop" to pause the timer
   - Click on stopped timers to edit inline (Enter or click away to save, Escape to cancel)
   - Click on task titles to edit title/description (Enter or click away to save, Shift+Enter for newline, Esc to cancel)
   - Click "✕" to delete entries (with confirmation)
   - Only one timer runs at a time

4. **Navigate between days:**
   - Click ‹ › buttons to go to previous/next day
   - Use arrow keys (← →) for keyboard navigation
   - Click on the date to open calendar picker
   - Click on any day in the weekly chart to jump to that day
   - Click "Today" button to return to current day

5. **Weekly overview:**
   - Visual bar chart shows hours worked each day of the week
   - Bar height indicates relative amount of work
   - Current day highlighted in green
   - Selected day highlighted in blue

## Architecture

- **Backend:** Go with Gorilla Mux router
- **Frontend:** HTML with HTMX for dynamic interactions
- **Styling:** Custom CSS with modern design
- **Data:** In-memory storage with date awareness (resets on restart)

## Project Structure

```
.
├── main.go           # Go backend server
├── go.mod           # Go module definition
├── templates/       # HTML templates
│   ├── index.html   # Main page template
│   └── time_entry.html # Time entry partial template
├── static/          # Static assets
│   └── style.css    # Application styles
└── README.md        # This file
```

## API Endpoints

- `GET /` - Redirects to today's date
- `GET /{date}` - Main application page for specific date (YYYY-MM-DD)
- `POST /{date}/add` - Add new time entry (auto-starts)
- `POST /{date}/start-stop/{id}` - Toggle timer for entry
- `GET /{date}/update-timer/{id}` - Get current timer value
- `POST /{date}/edit-time/{id}` - Update timer duration
- `POST /{date}/edit-description/{id}` - Update entry description
- `DELETE /{date}/delete/{id}` - Delete time entry

## Next Steps

Future enhancements could include:
- Database persistence
- Monthly/yearly summary views
- User authentication
- Export functionality (CSV, PDF reports)
- Project categorization and tagging
- Time goals and targets
- Detailed time analytics and reporting
