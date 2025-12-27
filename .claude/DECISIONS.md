# Decisions Log - hub-tui

## 2025-12-27: Inline Modals Instead of Overlay

**Decision:** Switch from centered overlay modals to inline modals (Claude Code style) that render between the chat area and status bar.

**Rationale:**
The original overlay approach required complex rendering with lipgloss.Place() to center the modal over the chat view. This caused persistent background color artifacts - lighter bars appearing to the right of text where the Surface background wasn't filling properly. Despite multiple attempts to fix with explicit Background() styles and Width() settings, the overlay approach remained visually inconsistent.

The inline approach is simpler: the modal is just another element in the vertical layout stack. It appears between messages and the status bar, similar to how Claude Code displays /plugin or /usage modals. This eliminates overlay rendering entirely and reuses the existing status bar for the Ctrl+C quit hint.

**Alternatives Considered:**
1. Keep fixing overlay backgrounds - Tried multiple approaches (explicit backgrounds on all styles, width-filling content wrappers) but artifacts persisted due to lipgloss rendering behavior with nested styles
2. Custom overlay rendering without lipgloss.Place() - Would require manual string manipulation to overlay content, adding complexity
3. Full-screen modal replacing chat view - Too disruptive, loses context of conversation

**Impact:**
- Simplifies `internal/ui/modal/modal.go` - removes overlay logic, SetCtrlCPressed, just returns rendered content
- Updates `internal/app/app.go` renderMain() - inserts modal between chat and status bar
- Input handling unchanged - keys already route to modal when open
- Future modals automatically get consistent styling
- May want to visually indicate input is disabled when modal open (future enhancement)

---
