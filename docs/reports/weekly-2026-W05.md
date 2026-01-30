# Weekly Report - hub-tui - Week of 2026-01-27 (W05)

## Week Overview
Exceptional progress this week, completing two major feature plans. Finished the v8-module-registry plan (admin module management) and made substantial progress on v9-workflow-builder, completing 7 of 8 phases. The workflow builder now supports full visual editing of automations including tool selection, step configuration, testing, and variable passing between steps.

## Key Accomplishments

### Module Registry (v8 - Completed)
- **Phase 1**: Client layer with registry API methods
- **Phase 2**: Modal refactor with view states, detail view, double-press disable
- **Phase 3**: Admin browse available modules, install from registry
- **Phase 4**: Admin uninstall with affected users confirmation
- **Phase 5**: Admin update with version indicators (⬆)
- Archived v8-module-registry plan

### Workflow Builder (v9 - Phases 1-7 Complete)
- **Phase 1**: Client layer - CRUD methods, builder API (GetBuilderTools, TestStep, PreviewSchedule)
- **Phase 2**: List view - [n]ew, [e]dit, [d]elete with double-press confirmation
- **Phase 3.1**: Builder state & display - step list with type/target/save_as
- **Phase 3.2**: Builder editing - name editing, output dropdown, [J/K] reorder steps
- **Phase 4**: Trigger form - schedule configuration with cron preview and next runs
- **Phase 5**: Tool picker - TreePicker component, Category → Source → Tool navigation
- **Phase 6**: Step detail form - dynamic params, profile dropdown for integrations
- **Phase 7.1**: Step testing - [t] test with variable substitution, output display
- **Phase 7.2**: Variable picker - [v] to pick fields from previous steps' outputs

### New Reusable Components
- `TreePicker` - Hierarchical navigation (j/k, Enter to drill down, Esc to go back)
- `FieldPicker` - JSON field extraction with smart heuristics for Notion properties
- `ScheduleForm` - Trigger configuration with frequency/time/days

### UX Improvements
- Cursor navigation in text fields (←/→, Home/End, Delete)
- Inline hints on param fields: `[Enter] Edit  [v] Variable`
- Profile dropdown with `◀ value ▶` navigation
- Focusable buttons replacing hotkeys (e.g., "+ Add step" instead of [a])

## Decisions Made
No formal decisions logged this week, but established patterns:
- Variable picker shows **previous steps' outputs** (not current step) - correct flow for step chaining
- `[v]` for Variable instead of `[p]` for Pick - better mnemonic
- Inline hints on param fields rather than global hint bar
- Direct insertion on variable pick rather than two-step pick-then-insert
- `[Ctrl+s]` for save to match patterns elsewhere

## Challenges Encountered
1. **Tool picker stuck on "Loading tools..."** - Async message routing needed to be placed BEFORE view-specific routing in Update()
2. **Esc jumping all the way to root** - Changed Esc to go back one level at a time in TreePicker
3. **Variable picker UX confusion** - Initially showed current step's output; refactored to show previous steps' outputs which is the actual use case

## Metrics
- Commits: 2 (4a5a30e, 47b5ee0) + significant uncommitted work
- New files created: 6 (treepicker.go, fieldpicker.go, workflows_builder.go, workflows_schedule.go, workflows_step.go, workflows_tools.go)
- Lines added: ~1500+ across new and modified files
- Plans completed: 1 (v8-module-registry)
- Plans in progress: 1 (v9-workflow-builder, 7/8 phases done)

## Next Week Priorities
1. **Hub-core changes**: Add `properties` field to ToolParam for nested object schemas
2. **Hub-core changes**: Add `items` field for array type schemas
3. **Phase 8**: Polish & Edge Cases - error handling, nested params UI
4. Complete and archive v9-workflow-builder plan
5. Commit all workflow builder work

## Looking Ahead
After completing the workflow builder, potential next features:
- OAuth config type (requires browser redirect flow)
- Rich array editing with add/remove buttons
- Workflow enable/disable toggle
- Refactor all text fields to use "[Enter to edit]" pattern
