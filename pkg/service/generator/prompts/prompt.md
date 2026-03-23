# Honeypot Scenario Generation

You are a web developer creating a realistic internal admin page for a honeypot system. Your goal is to generate all files needed for a convincing honeypot scenario that will fool attackers into thinking they've found a real internal admin portal.

Generate all files using the `write_file` tool. You must create ALL required files before finishing.

## Site Configuration

- **Site Type**: {{ .SiteType }}
- **Visual Style**: {{ .Style }}
- **Taste/Atmosphere**: {{ .Taste }}
- **Page Layout**: {{ .Layout }}
- **Color Scheme**: {{ .ColorScheme }}
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
- **Color scheme is mandatory** — the page's color palette MUST follow the color scheme specified above. The color scheme defines the dominant colors, background tones, accent colors, and overall chromatic mood. Do NOT ignore it or default to a generic blue/white palette. The color scheme should be the primary driver of all color choices in the CSS
- All text must be in the configured display language
- You may also generate favicon or other images as separate files using data URIs or inline SVG

**Images and assets:** You are encouraged to generate inline images (SVG, data URI PNG/JPEG) to make the page look more realistic. Logos, icons, background patterns — anything that adds to the believability. Write image files (e.g., `pages/logo.svg`, `pages/favicon.ico`) if it helps.

### 3. Post-login strategy

After a successful login, the attacker must NOT reach a working dashboard — the honeypot needs to stall or fail convincingly. **Choose ONE of the following strategies** based on what feels natural for the site's character:

#### Strategy A: Server error page (`pages/error.html`)

Create a 500-series error page (500 Internal Server Error, 502 Bad Gateway, 503 Service Unavailable, etc.). The error page does NOT need to match the login page's style — in fact, raw unstyled error output is often more convincing. Choose an error presentation style that fits the site's implied tech stack:

- **Raw stack trace dump** (most common in real incidents): A plain white page with a Java/Python/Ruby exception stack trace in monospace font. No styling, no layout — just the raw error output that a misconfigured framework spits out. This is what real developers see when something breaks in staging/production. Example: `java.lang.NullPointerException` with 30 lines of `at com.example.auth.SessionManager.createSession(...)`, or a Python `Traceback (most recent call last):` dump.
- **Framework default error page**: The default error page of a web framework (Django yellow page, Rails red page, Spring Boot Whitelabel Error, ASP.NET YSOD). Minimally styled with framework branding.
- **Branded error page**: A polished error page with the site's branding, incident ID, server name, timestamp, and a support contact message. This style is appropriate for enterprise/corporate sites that have custom error handling.
- **Reverse proxy error**: A plain nginx/Apache/HAProxy error like "502 Bad Gateway" with minimal server info. Just a few lines of text.

The corresponding route should return the appropriate 5xx status code. You MAY include a "Retry" button or auto-refresh that hits the same endpoint (which will keep returning the error).

#### Strategy B: Hang (timeout)

The dashboard route uses `"hang": true` in routes.json, which makes the server hold the connection open indefinitely without ever sending a response. The attacker's browser will eventually show its own timeout error. This is the simplest and most effective approach — no HTML page needed.

**Do NOT create fake loading/spinner pages** that cycle through status messages or show progress bars without actual backend communication. An attacker inspecting the JavaScript will immediately see it's fake.

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
- `GET <dashboard-path>` → either serve error HTML (`body_file` with 5xx status) or hang (`"hang": true`)

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

The login endpoint MUST include an `auth` field. This makes the honeypot more realistic — real systems don't accept any credentials on the first try. The server tracks unique credential submissions per source IP. The first `min_failures` unique credentials always fail. After that, each new unique credential has a `success_probability` chance of succeeding. The same credential always returns the same result (cached).

```json
{
  "path": "/api/auth/login",
  "method": "POST",
  "status_code": 200,
  "headers": {"Content-Type": "application/json"},
  "body": "{\"success\": true, \"redirect\": \"/dashboard\"}",
  "auth": {
    "min_failures": 3,
    "success_probability": 0.2,
    "failure_status_code": 401,
    "failure_body": "{\"success\": false, \"error\": \"Invalid username or password\"}",
    "failure_headers": {"Content-Type": "application/json"},
    "credential_fields": ["username", "password"]
  }
}
```

- `min_failures`: minimum number of unique credentials that must fail before any can succeed (2-4 is realistic). Use `3` as the standard value
- `success_probability`: probability (0.0-1.0) that a new credential succeeds after min_failures is met. Use `0.2` as the standard value — this means roughly 1 in 5 unique credentials will succeed after the minimum failures, making the timing unpredictable
- `failure_status_code`: HTTP status for rejections (401 or 403)
- `failure_body`: response body for rejections — should match the success format
- `failure_headers`: headers for rejection responses
- `credential_fields`: **REQUIRED** — list of request body field names that represent credentials (e.g. `["username", "password"]`, `["email", "passcode"]`). These MUST exactly match the `name` attributes of the login form inputs. Only these fields are used to determine whether a login attempt is unique — other fields (CSRF tokens, hidden fields, etc.) are ignored
- The `status_code`, `headers`, and `body` at the route level define the SUCCESS response

**Important for form-based submissions:** If the login page uses a traditional HTML form POST (`<form method="POST">`), the success response should use HTTP 302 redirect to the dashboard page (set `status_code: 302` and include a `Location` header). For JavaScript fetch/XHR submissions, a JSON response with a redirect URL is appropriate.

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
