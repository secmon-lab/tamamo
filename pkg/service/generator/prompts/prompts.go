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
