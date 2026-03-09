<div align="center">
  <img src="static/logo.svg" alt="Punchcard Logo" height="270">
  
  # Punchcard - Daily Time Tracker
  
  A simple daily time tracking webapp built with Go and HTMX.
</div>

## Screenshots

<div align="center">
  
| Daily View | Monthly Overview |
|------------|------------------|
| ![Daily Time Tracking](https://github.com/user-attachments/assets/bffffce2-9144-4367-9d72-7f1b98afc7f8) | ![Monthly Calendar](https://github.com/user-attachments/assets/9ef87173-364f-422a-9064-6f2faeb32850) |
| *Track your daily tasks with real-time timers* | *Visual monthly overview with calendar grid* |

</div>

## Architecture

- **Backend:** Go with Gorilla Mux router
- **Frontend:** HTML with HTMX for dynamic interactions
- **Styling:** Custom CSS with modern design
- **Data:** SQLite database storage (data persists between restarts)

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
- Dark mode theme
