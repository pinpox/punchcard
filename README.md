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
- **Kimai JSON export**: Times can be exported as Kimai-compatible JSON (see [Importing to Kimai](#importing-to-kimai))
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

### Importing to Kimai

Download the JSON export from the punchcard UI (daily or monthly), then import
each entry to Kimai using curl:

```bash
export KIMAI_API_TOKEN="<your token>"
export KIMAI_INSTANCE="https://your-kimai-instance"
export PUNCHCARD_EXPORT="punchcard-export-2026-03.json"

jq -c '.[]' $PUNCHCARD_EXPORT | while IFS= read -r entry; do
  curl -s -X POST "$KIMAI_INSTANCE/api/timesheets" \
    -H "Authorization: Bearer $KIMAI_API_TOKEN" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    -d "$entry"
  echo
done
```

To create an API token, go to your Kimai profile page → **API** tab → create a
new token.

> **Note:** Kimai has no deduplication - importing the same file twice will
> create duplicate entries. The `billable` field requires the `edit_billable`
> permission in Kimai; if your user lacks it, strip the field with
> `jq -c '.[] | del(.billable)'` instead.
