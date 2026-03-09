# Punchcard - Daily Time Tracker

A simple daily time tracking webapp built with Go and HTMX.

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

- **Backend:** Go with Gorill<div align="center">
  <img src="static/favicon.svg" alt="Punchcard Logo" width="96" height="96">
  
  # Punchcard - Daily Time Tracker
  
  A simple daily time tracking webapp built with Go and HTMX.
</div>

## Features

- ✅ Daily time tracking with date display (shows current day)
- ✅ Monthly overview with calendar grid and statistics
- ✅ Weekly bar chart showing hours worked each day (smart scaling for small durations)
- ✅ Previous/Next day and month navigation with keyboard shortcuts (← →)
- ✅ Calendar date picker (click on date to open)
- ✅ Quick "Today" button when viewing other dates
- ✅ Date validation with leap year support (automatically handles invalid dates)
- ✅ SQLite database persistence with Unix timestamps (no timezone issues)
- ✅ Add new time tracking tasks (auto-start timers)
- ✅ Start/Stop timers for each task
- ✅ Quick inline time editing (click timer, type "2", "0.25", or "1:30", Enter to save)
- ✅ Decimal hours support (e.g., "0.25" for 15 minutes)
- ✅ Auto-expanding editable titles and descriptions (click away to save, no scrollbars)
- ✅ Delete entries with confirmation
- ✅ Single timer mode (only one running at a time per day)
- ✅ Real-time timer updates (every second)
- ✅ Clean, minimal UI focused on the essentials
- ✅ Keyboard shortcuts for navigation (arrow keys)
- ✅ No page refreshes (HTMX powered)
- ✅ Custom favicon with punchcard theme

## Screenshots

<div align="center">
  
| Daily View | Monthly Overview |
|------------|------------------|
| ![Daily Time Tracking](screenshots/daily-view.png) | ![Monthly Calendar](screenshots/monthly-view.png) |
| *Track your daily tasks with real-time timers* | *Visual monthly overview with calendar grid* |

</div>

## Quick Start

1. **Run the application:**
   ```bash
   go run main.go
   ```

2. **Open your browser:**
   Navigate to `http://localhost:8080` (redirects to today)
   Or go directly to a specific day: `http://localhost:8080/2025-03-11`
   
   **Note:** Invalid dates like `2023-02-29` (non-leap year) will show an error

3. **Database:**
   The application creates `punchcard.db` automatically on first run
   - All time entries are stored persistently
   - To backup your data, simply copy the `punchcard.db` file
   - To reset all data, delete the `punchcard.db` file

4. **Start tracking time:**
   - Add a new task using the input field (automatically starts timer)
   - Click "Start" to begin timing existing tasks
   - Click "Stop" to pause the timer
   - Click on stopped timers to edit inline (Enter or click away to save, Escape to cancel)
   - Click on task titles to edit title/description (Enter or click away to save, Shift+Enter for newline, Esc to cancel)
   - Click "✕" to delete entries (with confirmation)
   - Only one timer runs at a time

5. **Navigate between days:**
   - Click ‹ › buttons to go to previous/next day
   - Use arrow keys (← →) for keyboard navigation
   - Click on the date to open calendar picker
   - Click on any day in the weekly chart to jump to that day
   - Click "Today" button to return to current day

6. **Weekly overview:**
   - Visual bar chart shows hours worked each day of the week
   - Smart scaling: small durations get appropriate bar heights (no more huge bars for tiny times)
   - Chart updates automatically when you start/stop timers or edit times
   - Current day highlighted in green
   - Selected day highlighted in blue

7. **Monthly overview:**
   - Click "Month" button to view calendar grid for the entire month
   - Navigate between months using ‹ › buttons or arrow keys
   - Each day shows total hours, entry count, and first task title
   - Monthly statistics: total hours, entries, and average hours per workday
   - Click any day to jump to detailed day view
   - Read-only view focused on overview and navigation

## Architecture

- **Backend:** Go with Gorilla Mux router
- **Frontend:** HTML with HTMX for dynamic interactions
- **Styling:** Custom CSS with modern design
- **Data:** SQLite database storage (data persists between restarts)

## Project Structure

```
.
├── main.go           # Go backend server with SQLite integration
├── go.mod           # Go module definition
├── punchcard.db     # SQLite database (created automatically)
├── templates/       # HTML templates
│   ├── index.html   # Main page template
│   ├── month.html   # Monthly overview template
│   └── time_entry.html # Time entry partial template
├── static/          # Static assets
│   ├── style.css    # Application styles
│   ├── favicon.svg  # App favicon (SVG)
│   └── favicon.ico  # App favicon (ICO)
├── assets/          # Source assets
│   └── original-logo.svg # Original logo file
├── screenshots/     # Application screenshots
└── README.md        # This file
```

## API Endpoints

- `GET /` - Redirects to today's date
- `GET /{date}` - Main application page for specific date (YYYY-MM-DD)
- `GET /month/{month}` - Monthly overview page (YYYY-MM format)
- `POST /{date}/add` - Add new time entry (auto-starts)
- `POST /{date}/start-stop/{id}` - Toggle timer for entry
- `GET /{date}/update-timer/{id}` - Get current timer value
- `POST /{date}/edit-time/{id}` - Update timer duration (supports decimal hours)
- `POST /{date}/edit-description/{id}` - Update entry description
- `DELETE /{date}/delete/{id}` - Delete time entry

## Next Steps

Future enhancements could include:
- Yearly summary views and trends
- User authentication and multi-user support
- Export functionality (CSV, PDF reports)
- Project categorization and tagging
- Time goals and targets
- Data backup and sync features
- Detailed time analytics and reporting
- REST API for integrations
- Mobile-responsive improvements
- Dark mode theme Mux router
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
