<div align="center">
  <img src="static/logo.svg" alt="Punchcard Logo" height="270">
  
  # Punchcard - Daily Time Tracker
  
  A simple daily time tracking webapp built with Go and HTMX.
</div>

<div align="center">
  
| Daily View | Monthly Overview |
|------------|------------------|
| ![Daily Time Tracking](https://github.com/user-attachments/assets/9ef87173-364f-422a-9064-6f2faeb32850) | ![Monthly Calendar](https://github.com/user-attachments/assets/bffffce2-9144-4367-9d72-7f1b98afc7f8) |
| *Track your daily tasks with real-time timers* | *Visual monthly overview with calendar grid* |

</div>

## Features

- **OIDC Authentication**: Secure login with any OIDC provider (Keycloak, Auth0, Google, etc.)
- **Billable Tracking**: Mark time entries as billable/non-billable with visual indicators
- **Kimai JSON export**: Times can be exported as Kimai-compatible JSON
- **Billable/Unbillable**: Each time entry has a billable checkbox

All time entries (both billable and non-billable) are included in exports with
correct status. Unbillable don't count towards the monthly and weekly total in
the UI.

### Configuration

Punchcard is configured via environment variables (`.env` file will be
automatically loaded).

```bash
PORT=8080                                    # Server port (default: 8080)
DATABASE_URL=punchcard.db                    # Database file path (default: punchcard.db)
OIDC_ISSUER_URL=https://your-provider.com    # OIDC provider URL
OIDC_CLIENT_ID=your-client-id                # OAuth client ID
OIDC_CLIENT_SECRET=your-client-secret        # OAuth client secret
OIDC_REDIRECT_URL=http://domain.tld/callback # OAuth redirect URL
```
