package prompts

import (
	"bytes"
	"crypto/rand"
	_ "embed"
	"math/big"
	"text/template"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/tamamo/pkg/utils/errutil"
)

//go:embed prompt.md
var basePromptTemplate string

// Params holds parameters for scenario generation prompt template.
type Params struct {
	SiteType    string
	Style       string
	Taste       string
	Layout      string
	Lang        string
	ExtraPrompt string
}

var (
	defaultSiteTypes = []string{
		// Business & operations
		"Purchase order management system",
		"Expense approval workflow portal",
		"Customer relationship management (CRM)",
		"Sales pipeline & forecasting dashboard",
		"Contract lifecycle management portal",
		"Vendor & supplier management system",
		"Invoice & accounts payable portal",
		"Travel & expense reporting system",
		"Meeting room & facility booking system",
		"Employee onboarding portal",
		"Shift scheduling & workforce management",
		"Internal helpdesk & ticketing system",
		"Document approval workflow",
		"Compliance training & certification tracker",
		"Quality assurance & inspection portal",

		// Industry-specific
		"Warehouse & logistics tracking system",
		"Fleet management dashboard",
		"Patient record management portal",
		"Clinical trial data management",
		"Building maintenance & work order system",
		"Equipment calibration tracking portal",
		"Retail POS configuration console",
		"Real estate property management portal",
		"Insurance claim processing system",
		"Restaurant reservation & table management",

		// HR & people
		"Employee directory & HR portal",
		"Performance review & appraisal system",
		"Payroll administration console",
		"Benefits enrollment portal",
		"Time & attendance tracking system",
		"Recruitment & applicant tracking system",

		// IT & infrastructure
		"IT asset management portal",
		"Infrastructure monitoring dashboard",
		"CI/CD pipeline console",
		"Cloud resource management panel",
		"Incident response console",
		"VPN / network access manager",

		// Data & reporting
		"Business intelligence dashboard",
		"Internal analytics & reporting portal",
		"Inventory management system",
		"Internal wiki / knowledge base",
		"Audit log viewer & compliance dashboard",
		"Budget planning & financial reporting tool",
	}
	defaultStyles = []string{
		"corporate-minimal",
		"dark-tech",
		"material-design",
		"bootstrap-classic",
		"flat-modern",
		"sidebar-nav",
		"dashboard-grid",
		"terminal-green",
		"glassmorphism",
		"neobrutalism",
		"soft-pastel",
		"monochrome-pro",
		"high-contrast-accessible",
		"retro-web2.0",
		"compact-dense",
	}
	defaultTastes = []string{
		// Outdated / neglected
		"Looks like it was built 10 years ago and never updated",
		"Clearly running on an old framework, has that early 2010s Bootstrap look",
		"Feels abandoned — copyright footer still says 2019",
		"Ugly and clunky, like an intern built it and nobody maintained it",
		"Has that classic 'enterprise Java webapp' feel with default styling",
		"Old-school table-based layout, probably migrated from an even older system",

		// Hastily built / amateur
		"Thrown together quickly — default fonts, minimal styling, no polish",
		"Looks like someone followed a tutorial and shipped it as-is",
		"Quick and dirty internal tool, probably no one outside the team uses it",
		"Built with a low-code tool, generic and template-looking",
		"Has placeholder text and default icons that were never customized",
		"Feels like a weekend hackathon project that accidentally went to production",

		// Overconfident / careless security
		"Has 'Admin Panel' in the page title with no branding — lazy setup",
		"Login page with no rate limiting notice, no CAPTCHA, feels wide open",
		"Shows server version info in the footer — carelessly configured",
		"Default admin credentials probably still work vibe",
		"Debug mode accidentally left on feel — shows stack traces in error pages",

		// Suspiciously simple
		"Just a plain login form on a white page, nothing else",
		"Minimal to the point of looking unfinished",
		"Single-page app that feels like it's hiding a lot behind the login",

		// Modern but sloppy
		"Uses modern frameworks but clearly copy-pasted from docs",
		"Trendy dark mode UI but inconsistent spacing and broken alignment",
		"SPA with loading spinners everywhere — over-engineered but under-tested",
		"Looks polished on the surface but has typos and broken links",
	}
	defaultLayouts = []string{
		// Classic / simple
		`centered-card: A single card/box centered horizontally and vertically on the page. Card has rounded corners, subtle shadow or border. Form fields inside the card. Background is a solid color or simple gradient. The most common pattern — think of a typical Bootstrap or Material UI login template.`,

		`centered-minimal: Extremely minimal — just a small logo and bare form fields on a plain white or black background. No card, no shadow, no decoration. Google sign-in or Apple ID style. Feels clean and intentional, not lazy. Generous whitespace.`,

		`stacked-full-width: Form fields and labels span the full width of the page or a very wide container. No card wrapper. Feels like an old intranet or legacy enterprise system — maybe a <table>-based layout. Minimal CSS. Looks like it was built in the early 2000s and never redesigned.`,

		// Split / asymmetric
		`split-screen: Page divided into two vertical halves (roughly 50/50 or 40/60). One side has branding — company logo, product illustration, tagline, decorative gradient, or a large background image. The other side has the login form. Common in modern SaaS and enterprise apps like Okta, Auth0, or Azure AD.`,

		`left-aligned: Login form is positioned on the left third of the page. Right side is mostly empty whitespace, or contains a subtle decorative element (illustration, pattern, or product screenshot). Asymmetric and intentional. Feels like a designer made this choice.`,

		`sidebar-form: A narrow sidebar (left or right, 250-350px wide) contains the login form. The main content area shows product information, company news feed, system status, or a welcome message. The sidebar feels like a persistent navigation element. Common in portal-style enterprise apps.`,

		// Layered / overlay
		`full-background: A full-viewport background image (server room, office, abstract tech pattern) or rich gradient covers the entire page. The login form floats on top, either in a semi-transparent card or with a backdrop-blur effect. The background sets the mood.`,

		`floating-panel: The background shows a blurred or dimmed screenshot of what the actual dashboard looks like — giving the attacker a preview of what's behind the login. A floating panel or card in the center/corner contains the login form. Creates anticipation.`,

		`dialog-modal: The background shows a partially visible page (documentation, portal homepage, public status page). A modal dialog overlay demands authentication before proceeding. Has a semi-transparent backdrop. May include a "close" button that does nothing or shows "authentication required" message.`,

		// Navigation-integrated
		`top-bar: A full-width navigation bar/header at the top of the page with the company logo, product name, and possibly links (Help, Status, Docs). The login form sits below in the main content area. Feels like an enterprise app where the login page shares the same chrome as the rest of the application.`,

		`compact-inline: The login form is embedded directly in a top navigation bar, a dropdown panel, or a collapsible section — not on a dedicated page. The page itself might show public content (API docs, status page, product info) with a small "Sign In" area tucked into the header or a slide-out panel.`,

		// Multi-step / wizard
		`wizard-flow: Login is broken into multiple sequential steps, each on its own screen or animated transition. Step 1 might ask for organization/tenant name, Step 2 for username/email, Step 3 for password. Include step indicators (dots, numbers, or breadcrumbs). Only show the current step's fields. Back button between steps.`,

		`vertical-stepper: A vertical stepper component on the left side shows numbered steps (e.g., 1. Authenticate → 2. Verify Identity → 3. Accept Terms). The right/main area shows the current step's form. Completed steps show checkmarks. Upcoming steps are greyed out. Enterprise SSO feel.`,

		`card-with-tabs: A single centered card with tab navigation at the top. Tabs switch between different authentication methods: "Credentials" (username/password), "SSO" (SAML button), "Certificate" (file upload or smart card prompt), "Token" (one-time code input). Only one tab's content is visible at a time.`,

		// Decorative / branded
		`hero-banner: Top portion of the page (30-40% height) is a large hero section with a banner image, product illustration, company branding, or a welcome message. The login form sits below the hero in a card or inline section. Common in SaaS products and customer-facing portals.`,

		`notification-bar: A standard login layout (any basic arrangement) with a persistent notification banner fixed to the top or bottom of the page. The banner shows a system notice: scheduled maintenance window, security advisory, version update announcement, or "Unauthorized access is monitored and logged" warning. The banner should feel urgent but routine.`,

		// Unconventional
		`terminal-cli: The entire page looks like a terminal or command-line interface. Monospace font, dark background (black or dark green), blinking cursor. Login prompts appear as "Username: " and "Password: " text lines. May include ASCII art logo, system banner message ("Welcome to ACME Corp Secure Shell"), or fake boot sequence text above the login prompt.`,

		`bottom-sheet: Mobile-app-inspired layout. The main page shows a branded splash or minimal content. The login form appears in a bottom sheet that slides up from the bottom of the viewport, covering the lower 50-60% of the screen. Has a drag handle at the top. Feels like a native mobile app running in a browser.`,

		`iframe-embedded: The login form appears to be embedded within a larger parent page. The parent page has its own header (with company logo, navigation links like "Home", "Products", "Support") and footer. The login area sits in the middle content zone, possibly with a border or slight inset that makes it look like an iframe or embedded component. Feels like a corporate website with a login widget.`,
	}
	defaultLang = "English"
)

// Resolve fills in any empty parameters with random defaults.
func (p *Params) Resolve() {
	if p.SiteType == "" {
		p.SiteType = defaultSiteTypes[cryptoRandN(len(defaultSiteTypes))]
	}
	if p.Style == "" {
		p.Style = defaultStyles[cryptoRandN(len(defaultStyles))]
	}
	if p.Taste == "" {
		p.Taste = defaultTastes[cryptoRandN(len(defaultTastes))]
	}
	if p.Layout == "" {
		p.Layout = defaultLayouts[cryptoRandN(len(defaultLayouts))]
	}
	if p.Lang == "" {
		p.Lang = defaultLang
	}
}

// cryptoRandN returns a cryptographically secure random int in [0, n).
func cryptoRandN(n int) int {
	v, _ := rand.Int(rand.Reader, big.NewInt(int64(n)))
	return int(v.Int64())
}

// Build renders the system prompt template with the given parameters.
func Build(params *Params) (string, error) {
	params.Resolve()

	tmpl, err := template.New("prompt").Parse(basePromptTemplate)
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse prompt template",
			goerr.T(errutil.TagInternal),
		)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, params); err != nil {
		return "", goerr.Wrap(err, "failed to execute prompt template",
			goerr.T(errutil.TagInternal),
		)
	}

	return buf.String(), nil
}
