package api

// WhoamiResponse is returned by GET /api/v1/cli/whoami.
// We surface enough info to confirm the user authenticated against the right
// org and to show it in `ops0 login` output.
type WhoamiResponse struct {
	UserID         string   `json:"userId"`
	UserEmail      string   `json:"userEmail"`
	OrganizationID string   `json:"organizationId"`
	Organization   string   `json:"organization"`
	APIKeyName     string   `json:"apiKeyName"`
	Scopes         []string `json:"scopes"`
}

// Policy is the shape returned by GET /api/v1/cli/policies.
//
// `Rego` is the actual Rego rule body — included so the CLI can run OPA
// locally without a follow-up roundtrip. For large policy sets we may move
// this to a separate `bundle` endpoint that returns a tarball; today the list
// is small enough that inlining is fine.
type Policy struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Description       string   `json:"description,omitempty"`
	Category          string   `json:"category"` // security | cost | compliance | tagging | best-practices | custom
	Severity          string   `json:"severity"` // error | warning | info
	Rego              string   `json:"rego"`
	PolicyType        string   `json:"policyType"` // iac | kubernetes
	IsEnabled         bool     `json:"isEnabled"`
	CoversTemplateIDs []string `json:"coversTemplateIds,omitempty"` // mapping to discovery security check IDs
}

type policiesResponse struct {
	Policies []Policy `json:"policies"`
}

// IaCStandards is the shape returned by GET /api/v1/cli/iac-standards.
//
// The CLI's MCP tool `get_iac_standards` forwards this response verbatim
// to the agent so it can ground its output on the same tag / doc /
// mermaid contract the ops0 web-UI IaC agent uses. When the server
// disagrees with the static CLAUDE.md baseline, the server wins - that's
// the whole reason the endpoint exists.
type IaCStandards struct {
	RequiredTags     []RequiredTag              `json:"requiredTags"`
	ProviderTagField map[string]ProviderTagInfo `json:"providerTagField"`
	Documentation    DocumentationContract      `json:"documentation"`
	MermaidRules     MermaidRules               `json:"mermaidRules"`
	Recommendations  []Recommendation           `json:"recommendations"`
	Scope            StandardsScope             `json:"scope"`
	Version          int                        `json:"version"`
}

type RequiredTag struct {
	Key               string   `json:"key"`
	Required          bool     `json:"required"`
	Description       string   `json:"description,omitempty"`
	SuggestedVariable string   `json:"suggestedVariable,omitempty"`
	FixedValue        string   `json:"fixedValue,omitempty"`
	AllowedValues     []string `json:"allowedValues,omitempty"`
}

type ProviderTagInfo struct {
	Field  string `json:"field"`  // "tags" / "labels" / "freeform_tags"
	Casing string `json:"casing"` // "PascalCase" / "lowercase"
	Note   string `json:"note,omitempty"`
}

type DocumentationContract struct {
	Readme          ReadmeSpec   `json:"readme"`
	ArchitectureDoc DocSpec      `json:"architectureDoc"`
	RunbookDoc      DocSpec      `json:"runbookDoc"`
}

type ReadmeSpec struct {
	Path     string   `json:"path"`
	Required bool     `json:"required"`
	Sections []string `json:"sections"`
}

type DocSpec struct {
	Path     string `json:"path"`
	When     string `json:"when"`
	Contents string `json:"contents"`
}

type MermaidRules struct {
	FenceFormat        string                   `json:"fenceFormat"`
	DirectionModifiers string                   `json:"directionModifiers"`
	ArchitectureBeta   ArchitectureBetaRules    `json:"architectureBeta"`
}

type ArchitectureBetaRules struct {
	Labels  string `json:"labels"`
	IDs     string `json:"ids"`
	Icons   string `json:"icons"`
	Edges   string `json:"edges"`
	Parents string `json:"parents"`
}

type Recommendation struct {
	ID                        string  `json:"id"`
	Category                  string  `json:"category"` // cost | security | drift | compliance
	Title                     string  `json:"title"`
	Detail                    string  `json:"detail"`
	Adopted                   bool    `json:"adopted"`
	EstimatedMonthlyImpactUSD float64 `json:"estimatedMonthlyImpactUsd,omitempty"`
}

type StandardsScope struct {
	ProjectID *string `json:"projectId"`
}

// CheckReport is the telemetry payload sent by `ops0 policies check` when
// telemetry is enabled. We never include source code — just policy IDs and
// pass/fail counts plus a hash of the file paths so the dashboard can group
// runs without learning anything sensitive.
type CheckReport struct {
	ProjectID  string             `json:"projectId,omitempty"`
	RepoHash   string             `json:"repoHash"`   // sha256 of the repo path (stable per machine)
	Total      int                `json:"total"`      // total checks run
	Passed     int                `json:"passed"`
	Failed     int                `json:"failed"`
	Violations []CheckViolation   `json:"violations,omitempty"`
	Duration   int                `json:"durationMs"`
	CLIVersion string             `json:"cliVersion"`
}

type CheckViolation struct {
	PolicyID    string `json:"policyId"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	FilePath    string `json:"filePath,omitempty"`
	LineStart   int    `json:"lineStart,omitempty"`
	Resource    string `json:"resource,omitempty"`
	Remediation string `json:"remediation,omitempty"`
}
