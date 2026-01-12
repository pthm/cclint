package rules

// Registry holds all registered rules
type Registry struct {
	rules []Rule
}

// NewRegistry creates a new rule registry
func NewRegistry() *Registry {
	return &Registry{
		rules: make([]Rule, 0),
	}
}

// Register adds a rule to the registry
func (r *Registry) Register(rule Rule) {
	r.rules = append(r.rules, rule)
}

// Rules returns all registered rules, optionally filtering by AI requirement.
// If includeAI is false, rules with RequiresAI=true are excluded.
func (r *Registry) Rules(includeAI bool) []Rule {
	if includeAI {
		return r.rules
	}

	var result []Rule
	for _, rule := range r.rules {
		if !rule.Config().RequiresAI {
			result = append(result, rule)
		}
	}
	return result
}

// Get returns a rule by name
func (r *Registry) Get(name string) Rule {
	for _, rule := range r.rules {
		if rule.Name() == name {
			return rule
		}
	}
	return nil
}

// DefaultRegistry returns a registry with all default rules
func DefaultRegistry() *Registry {
	r := NewRegistry()

	// Register structural rules
	r.Register(&BrokenRefsRule{})
	r.Register(&CircularRefsRule{})
	r.Register(&LongDocumentRule{})
	r.Register(&MissingEntrypointRule{})
	r.Register(&BroadPermissionsRule{})
	r.Register(&DuplicateInstructionsRule{})
	r.Register(&MissingToolRule{})
	r.Register(&MissingSkillRule{})

	// Register content quality rules
	r.Register(&VagueInstructionsRule{})
	r.Register(&ContradictionsRule{})
	r.Register(&MissingContextRule{})
	r.Register(&VerbosityRule{})

	// Register AI rules (requires --deep flag)
	// These rules analyze configurations per-scope (main agent and each subagent separately)
	if rule := NewLLMDuplicatesRule(); rule != nil {
		r.Register(rule)
	}
	if rule := NewLLMContradictionsRule(); rule != nil {
		r.Register(rule)
	}
	if rule := NewLLMClarityRule(); rule != nil {
		r.Register(rule)
	}
	if rule := NewLLMActionabilityRule(); rule != nil {
		r.Register(rule)
	}

	return r
}
