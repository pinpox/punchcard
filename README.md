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

## Multi-User Support & Authentication

Punchcard now supports multi-user environments with **OpenID Connect (OIDC)** authentication:

### Features
- **OIDC Authentication**: Secure login with any OIDC provider (Keycloak, Auth0, Google, etc.)
- **User Isolation**: Each user's time entries are completely isolated
- **Session Management**: Secure session handling with automatic expiry
- **User Profiles**: View personal statistics and account information
- **Auto-configuration**: Automatically loads `.env` files for easy setup
- **Billable Tracking**: Mark time entries as billable/non-billable with visual indicators

### Setup

1. **Clone and build:**
   ```bash
   git clone <repository-url>
   cd punchcard
   go mod tidy
   go build
   ```

2. **Configure OIDC Authentication:**
   ```bash
   cp .env.example .env
   # Edit .env with your OIDC provider settings
   ```

3. **Run the application:**
   ```bash
   # Automatically loads .env file
   ./punchcard
   ```

### Environment Variables

#### Required for OIDC Authentication
```bash
OIDC_ISSUER_URL=https://your-provider.com        # OIDC provider URL
OIDC_CLIENT_ID=your-client-id                    # OAuth client ID
OIDC_CLIENT_SECRET=your-client-secret            # OAuth client secret
OIDC_REDIRECT_URL=http://localhost:8080/callback # OAuth redirect URL
```

#### Optional Configuration
```bash
PORT=8080                    # Server port (default: 8080)
DATABASE_URL=punchcard.db    # Database file path (default: punchcard.db)
```

### OIDC Provider Examples

#### Keycloak
```bash
OIDC_ISSUER_URL=https://keycloak.example.com/auth/realms/myrealm
OIDC_CLIENT_ID=punchcard
OIDC_CLIENT_SECRET=your-keycloak-client-secret
OIDC_REDIRECT_URL=http://localhost:8080/callback
```

#### Auth0
```bash
OIDC_ISSUER_URL=https://your-tenant.auth0.com/
OIDC_CLIENT_ID=your-auth0-client-id
OIDC_CLIENT_SECRET=your-auth0-client-secret
OIDC_REDIRECT_URL=http://localhost:8080/callback
```

#### Google OAuth
```bash
OIDC_ISSUER_URL=https://accounts.google.com
OIDC_CLIENT_ID=your-google-client-id.apps.googleusercontent.com
OIDC_CLIENT_SECRET=your-google-client-secret
OIDC_REDIRECT_URL=http://localhost:8080/callback
```

#### Gitea
```bash
OIDC_ISSUER_URL=https://gitea.example.com/
OIDC_CLIENT_ID=your-gitea-client-id
OIDC_CLIENT_SECRET=your-gitea-client-secret
OIDC_REDIRECT_URL=http://localhost:8080/callback
```

### Database Schema

The application automatically creates the required database tables on first run:
- **users**: OIDC user information and profiles
- **sessions**: Secure session management
- **time_entries**: Time tracking data with user isolation

> **Note**: OIDC authentication is required. The application will not start without proper OIDC configuration.

### Routes

#### Authentication Routes
- `GET /login` - Login page
- `GET /login/start` - Initiate OIDC flow
- `GET /callback` - OIDC callback handler
- `GET /logout` - Sign out
- `GET /profile` - User profile page

#### Application Routes (authenticated)
- `GET /` - Redirects to today's date
- `GET /{date}` - Main application page for specific date (YYYY-MM-DD)
- `GET /month/{month}` - Monthly overview page (YYYY-MM format)
- `GET /month/{month}.json` - Export month data in Kimai JSON format
- `GET /{date}.json` - Export single day data in Kimai JSON format
- All existing API endpoints (scoped to authenticated user)

### Kimai Export

Export your time tracking data in Kimai-compatible JSON format:

- **Monthly export**: Visit `/month/2024-03.json` to download March 2024 data
- **Daily export**: Visit `/2024-03-15.json` to download March 15, 2024 data

**Export format:**
```json
[
  {
    "project": 1,
    "activity": 1, 
    "begin": "2024-03-15T09:00:00",
    "end": "2024-03-15T17:30:00",
    "description": "Task description",
    "billable": true,
    "exported": false
  }
]
```

**Note**: You'll need to modify the `project` and `activity` IDs to match your Kimai instance before importing.

### Billable Time Tracking

Each time entry has a billable checkbox with visual indicators:

- **Green $ symbol**: Billable time (counts toward totals)
- **Empty checkbox**: Non-billable time (excluded from totals)

**Features:**
- **Click to toggle**: Simple click toggles billable status
- **Visual feedback**: Green dollar sign for billable, empty for non-billable
- **Smart totals**: Week/month hour totals only include billable time
- **Export ready**: Billable status included in Kimai export format

Only billable hours are counted in:
- Header week/month totals
- Profile statistics  
- Calendar view summaries

All time entries (both billable and non-billable) are included in exports with correct status.

## Next Steps

Future enhancements could include:
- Yearly summary views and trends
- Export functionality (CSV, PDF reports)
- Project categorization and tagging
- Time goals and targets
- Data backup and sync features
- Detailed time analytics and reporting
- REST API for integrations
- Mobile-responsive improvements
- Dark mode theme
- Team/organization features
- Advanced reporting and insights
