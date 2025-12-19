# Evaluation Cache Index
**Last Updated**: 2025-12-18

## Cached Knowledge Files

### project-structure.md
- **Confidence**: FRESH
- **Content**: Project layout, package responsibilities, key patterns
- **Reuse For**: Understanding codebase organization, where to find components
- **Invalidate When**: Major refactoring, package restructuring

### test-infrastructure.md
- **Confidence**: FRESH
- **Content**: Test commands, coverage analysis, test patterns
- **Reuse For**: Running tests, understanding what's tested, adding new tests
- **Invalidate When**: Test framework changes, new test patterns introduced

## How to Use This Cache

### For Future Evaluations
1. Check INDEX.md for relevant cached knowledge
2. Verify confidence level (FRESH/RECENT/RISKY/STALE)
3. Reuse cached info instead of re-discovering
4. Update cache if you find new patterns

### Confidence Levels
- **FRESH**: Just created, fully accurate
- **RECENT**: <1 week old, likely still accurate
- **RISKY**: 1-4 weeks old, verify before using
- **STALE**: >4 weeks old, re-evaluate before using

### When to Update Cache
- New architectural patterns discovered
- Test infrastructure changes
- Coverage patterns shift
- New components added

## Cache Strategy

This cache stores **stable, reusable knowledge**:
- ✅ Project structure and organization
- ✅ Test commands and infrastructure
- ✅ Common patterns and conventions
- ❌ Specific test results (those go in WORK-EVALUATION files)
- ❌ Bug reports (those go in WORK-EVALUATION files)
- ❌ Point-in-time verdicts (those go in WORK-EVALUATION files)
