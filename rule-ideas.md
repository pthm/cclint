Core:

- Config tree building - parsing CLAUDE.md, .claude/, .mcp.json, settings.json, etc.
- Reference extraction - @ mentions, file paths, URLs, tool references
- Quality scoring - length, structure, clarity metrics
- Subagent/skill analysis - integration checking

Additions:

Validation & Correctness

- Broken reference detection (missing files, dead URLs, non-existent tools)
- Circular reference detection in subagent chains
- MCP server config validation (do referenced servers/tools exist?)
- Hook validation (do referenced scripts exist and are they executable?)

Security & Hygiene

- Secret/credential scanning (API keys, tokens accidentally in configs)
- Permission scope analysis (what's being granted?)
- Overly broad glob patterns in settings

Optimization Metrics

- Token/context estimation (how much context budget do configs consume?)
- Duplication detection across files (redundant instructions)
- Instruction conflict detection (contradictory rules)

Best Practices

- Known anti-patterns (overly verbose, unclear instructions, missing sections)
- Coverage gaps (no error handling guidance, no style guidance, etc.)
- Comparison against "gold standard" templates

Developer Experience

- Unused configs (defined but never referenced)
- Stale references (files that have moved/changed)
- Suggest consolidation opportunities
