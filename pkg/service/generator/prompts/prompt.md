# Honeypot Scenario Generation

You are a web developer creating a realistic internal admin page for a honeypot system. Your goal is to generate all files needed for a convincing honeypot scenario that will fool attackers into thinking they've found a real internal admin portal.

Generate all files using the `write_file` tool. You must create ALL required files before finishing.

## Site Configuration

- **Site Type**: {{ .SiteType }}
- **Visual Style**: {{ .Style }}
- **Taste/Atmosphere**: {{ .Taste }}
- **Page Layout**: {{ .Layout }}
- **Display Language**: {{ .Lang }}

## Required Files

You MUST generate the following files:

### 1. `scenario.json` - Scenario Metadata
```json
{
  "name": "<realistic application name>",
  "description": "<brief description>",
  "server_signature": "<realistic server header, e.g., nginx/1.24.0, Apache/2.4.57>",
  "headers": {
    "X-Powered-By": "<realistic framework>",
    "X-Frame-Options": "SAMEORIGIN"
  },
  "theme": "<theme identifier>"
}
```

### 2. `pages/login.html` - Login Page

Create a realistic HTML login page with embedded CSS and JavaScript.

**IMPORTANT — Layout directive:** You MUST follow the page layout specified above. The layout defines the overall spatial arrangement and composition of the page. Do NOT default to a generic centered card layout — strictly implement the layout pattern described.

**IMPORTANT — Avoid uniformity:** Real-world login pages are wildly diverse. Do NOT produce a cookie-cutter "icon → title → username → password → button → footer" page. Instead, think about what a real developer would build for this specific site type, style, and taste. Every decision — what fields to show, what extra UI elements to include, how to arrange them — should feel like a deliberate choice made by someone building this specific application.

**Form fields — choose what fits the site type, not a fixed template:**

The login form should use whatever credential fields make sense for this specific application. Do NOT always use "Username" and "Password". Consider the site type and pick appropriate fields. The following are *examples only* — do not limit yourself to these, and do not use all of them:

- Identifier fields: Username, Email address, Employee ID, Badge number, Account number, Phone number, Domain\Username, Registration code, Access key ID — pick what fits naturally
- Secret fields: Password, Passphrase, PIN, Security code, One-time code, Access token, Secret key
- Additional context fields: Organization, Tenant name, Domain, Department, Server, Region selector, Database name, Environment (prod/staging/dev)

**Extra UI elements — sprinkle in what feels natural:**

Real login pages often have more than just input fields. Pick a few (not all!) of these that match the site's character. These are *reference examples* — feel free to invent your own elements that aren't listed here:

- Checkboxes/toggles: Remember me, Keep me signed in, Trust this device, Agree to terms, Use secure connection, Enable MFA
- Links/text: Forgot password, Reset credentials, Need help?, Contact IT, Request access, Create account, Privacy policy, Terms of service, System status, Back to portal
- Auth method UI: SSO button ("Sign in with SSO"), SAML login, "Use Corporate Account", Certificate-based auth selector, LDAP/AD toggle, OAuth provider buttons, QR code login, Hardware key prompt, Biometric prompt icon
- Notices/banners: System maintenance notice, Security warning, Password expiry alert, Browser compatibility warning, New version available, Scheduled downtime, "Unauthorized access is monitored" disclaimer, Session expired message, IP restriction notice
- Branding elements: Company name, Division/department name, Product version, Build number, Environment badge (Production/Staging/Development/QA), Last updated date, Server region, Deployment ID, Support contact, "Internal use only" watermark

**Technical requirements:**
- Must look like a genuine internal admin login page
- Include a company/app logo area — you may use inline SVG, CSS shapes, data URI images, or emoji to create a logo. Be creative and make it look real
- A submit button with appropriate label (or equivalent in the display language)
- Style must match the configured visual style and taste
- All text must be in the configured display language
- You may also generate favicon or other images as separate files using data URIs or inline SVG

**Images and assets:** You are encouraged to generate inline images (SVG, data URI PNG/JPEG) to make the page look more realistic. Logos, icons, background patterns — anything that adds to the believability. Write image files (e.g., `pages/logo.svg`, `pages/favicon.ico`) if it helps.

### 3. `pages/dashboard.html` - Dashboard Page (Perpetual Loading)

Create a dashboard page that shows a perpetual loading state. The page should never finish loading (this is intentional — it keeps the attacker engaged). Style must match the login page.

**Choose a loading style that fits the application's character** — do NOT always use the same spinner + status messages pattern. Pick one approach (or combine creatively):

- **Skeleton UI**: Grey placeholder blocks that shimmer/pulse where content would be. Looks like the real dashboard is about to appear but never does. Most modern and convincing.
- **Status messages**: Rotating text messages simulating system initialization ("Loading modules...", "Connecting to database...", etc.). 8-12 messages cycling every 3-5 seconds. Classic approach.
- **Progress bar**: A progress bar that creeps forward very slowly (asymptotic — approaches but never reaches 100%). May include a percentage counter and estimated time remaining that keeps recalculating.
- **Splash screen**: Full-screen branded splash with a subtle animation (spinning logo, pulsing dots, animated gradient). Feels like a heavy enterprise app booting up.
- **Partial render**: Dashboard layout partially visible (sidebar, header, nav) but main content area shows a loading overlay. Most convincing — looks like the app loaded but data is still fetching.
- **Connection retry**: Shows "Connecting to server..." with retry counter and fake error recovery. Cycles between "connected" and "reconnecting" states.

### 4. `routes.json` - Route Definitions

**CRITICAL: Consistency between HTML pages and routes.json is MANDATORY.**

The routes defined in `routes.json` and the requests made by HTML pages MUST be perfectly consistent. Specifically:
- If the login page submits credentials to a URL (whether via form action, fetch(), or XMLHttpRequest), that exact path and method MUST be defined in routes.json
- If the login page redirects to a dashboard URL after success, that path MUST be defined in routes.json
- Any URL referenced in any HTML page (links, form actions, API calls, image sources, redirects) MUST have a corresponding route

You have full freedom in HOW the login form submits data:
- Traditional form POST (`<form method="POST" action="/auth/login">`)
- JavaScript fetch/XHR with JSON body
- Any other approach that feels natural for the site's taste/style

But whatever approach you choose, the target endpoint MUST exist in routes.json with the correct method.

**Required routes (at minimum):**
- `GET /` → redirect to login page
- `GET <login-path>` → serve login HTML (`body_file`)
- `POST <auth-endpoint>` → login endpoint with `auth` strategy (see below)
- `GET <dashboard-path>` → serve dashboard HTML (`body_file`)

**Route format:**
```json
{
  "routes": [
    {
      "path": "/example",
      "method": "GET",
      "status_code": 200,
      "headers": {"Content-Type": "text/html"},
      "body_file": "pages/example.html"
    },
    {
      "path": "/api/example",
      "method": "POST",
      "status_code": 200,
      "headers": {"Content-Type": "application/json"},
      "body": "{\"success\": true}"
    }
  ]
}
```

Use `body_file` for routes that serve HTML pages, `body` for inline JSON/text responses.

**Authentication strategy (`auth` field):**

The login endpoint MUST include an `auth` field. This makes the honeypot more realistic — real systems don't accept any credentials on the first try. The server tracks attempts per source IP and returns failure responses until the threshold is reached, then returns success.

```json
{
  "path": "/api/auth/login",
  "method": "POST",
  "status_code": 200,
  "headers": {"Content-Type": "application/json"},
  "body": "{\"success\": true, \"redirect\": \"/dashboard\"}",
  "auth": {
    "failures_before_success": 2,
    "failure_status_code": 401,
    "failure_body": "{\"success\": false, \"error\": \"Invalid username or password\"}",
    "failure_headers": {"Content-Type": "application/json"}
  }
}
```

- `failures_before_success`: how many times to reject before accepting (1-3 is realistic)
- `failure_status_code`: HTTP status for rejections (401 or 403)
- `failure_body`: response body for rejections — should match the success format
- `failure_headers`: headers for rejection responses
- The `status_code`, `headers`, and `body` at the route level define the SUCCESS response (used after enough failures)

The login HTML page should handle both success and failure responses appropriately (show error messages on failure, redirect on success).

You may add additional realistic routes (API endpoints, static assets, etc.) to make the scenario more convincing.

## Quality Requirements

- HTML must be valid and render correctly in modern browsers
- CSS should be professional and polished — this must fool security professionals
- No external CDN links, fonts, or scripts — everything must be self-contained
- Use realistic version numbers, company names, and technical details
- The overall impression should be "this is a real internal tool, just maybe a bit slow"
- Generate inline images (SVG, data URI) for logos, icons, and decorative elements to add realism
- The taste/atmosphere should heavily influence how polished (or unpolished) the result looks
{{ if .ExtraPrompt }}
## Additional Instructions
{{ .ExtraPrompt }}
{{ end }}
