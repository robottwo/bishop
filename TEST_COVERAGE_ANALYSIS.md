# Test Coverage Analysis

## Summary

Bishop is a Go-based generative shell application with ~184 Go source files and 75 test files. Based on the analysis, the overall test file ratio is approximately 40.8%, with 19 out of 28 packages having at least some test coverage.

## Coverage by Package

### Well-Tested Packages (>60% Coverage)

| Package | Coverage | Notes |
|---------|----------|-------|
| `pkg/reverse` | 100% | Simple utility, fully covered |
| `internal/bash` | 74.4% | Core shell parsing well tested |
| `internal/completion` | 72.5% | Completion system robust |
| `internal/environment` | 70.8% | Good coverage |
| `pkg/shellinput` | 65.7% | Shell input handling solid |
| `pkg/gline` | 61.1% | Line editing covered |

### Low-Coverage Packages (<50%)

| Package | Coverage | Notes |
|---------|----------|-------|
| `pkg/debounce` | 47.1% | Timing-sensitive code |
| `internal/utils` | 22.5% | Missing tests for LLM client |

### Packages with Zero Test Coverage

| Package | Files | Lines | Priority |
|---------|-------|-------|----------|
| `internal/predict` | 6 | ~427 | **HIGH** |
| `internal/config` | 1 | ~694 | **HIGH** |
| `internal/git` | 1 | ~129 | MEDIUM |
| `internal/system` | 5 | ~200 | MEDIUM |
| `internal/idle` | 1 | ~121 | MEDIUM |
| `internal/filesystem` | 1 | ~37 | LOW |
| `internal/styles` | 1 | ~50 | LOW |
| `internal/termfeatures` | 2 | ~100 | LOW |
| `internal/termtitle` | 1 | ~50 | LOW |
| `internal/rag` | 2 | ~100 | LOW |

---

## High-Priority Test Improvements

### 1. Prediction Engine (`internal/predict`) - **CRITICAL**

**Current State:** 0% coverage across 6 files (~427 lines)

**Why it matters:** This is the core AI/LLM prediction engine that powers the shell's intelligent command suggestions. Bugs here directly impact user experience.

**Files to test:**
- `predictrouter.go` - Router logic for predictions
- `prefix.go` - Prefix-based command prediction
- `nullstate.go` - Null state prediction
- `explainer.go` - Command explanation generation
- `schema.go` - JSON schema definitions

**Recommended tests:**
```go
// predictrouter_test.go
func TestPredictRouter_Predict_SkipsBlankInput(t *testing.T)
func TestPredictRouter_Predict_CallsPrefixPredictor(t *testing.T)
func TestPredictRouter_UpdateContext_UpdatesBothPredictors(t *testing.T)

// prefix_test.go
func TestLLMPrefixPredictor_Predict_SkipsAgentMessages(t *testing.T)
func TestLLMPrefixPredictor_Predict_IncludesHistoryContext(t *testing.T)
func TestLLMPrefixPredictor_UpdateContext_SetsContextText(t *testing.T)

// explainer_test.go
func TestLLMExplainer_Explain_ReturnsEmptyForEmptyInput(t *testing.T)
func TestLLMExplainer_Explain_FormatsPromptCorrectly(t *testing.T)
```

**Testing strategy:** Use mock LLM clients (interface injection) to test prompt construction and response parsing without making actual API calls.

---

### 2. Configuration UI (`internal/config`) - **HIGH**

**Current State:** 0% coverage for ~694 lines

**Why it matters:** Configuration management affects how users set up and persist their preferences. Bugs here can corrupt user settings.

**Key functions to test:**
- `GetSessionOverride` - Session config retrieval
- `saveConfig` - Config persistence to file
- `getEnv` - Environment variable reading with fallbacks
- `homeDir` - Home directory detection
- `handleSettingAction` - Setting type dispatch
- Model `Update` and `View` - TUI state machine

**Recommended tests:**
```go
// ui_test.go
func TestGetSessionOverride_ReturnsValueWhenSet(t *testing.T)
func TestGetSessionOverride_ReturnsFalseWhenNotSet(t *testing.T)
func TestGetEnv_ChecksSessionOverridesFirst(t *testing.T)
func TestGetEnv_FallsBackToRunnerVars(t *testing.T)
func TestSaveConfig_ValidatesInputBeforeSaving(t *testing.T)
func TestSaveConfig_HandlesSafetyChecksSpecially(t *testing.T)
func TestHandleSettingAction_TogglesToggleTypes(t *testing.T)
func TestHandleSettingAction_ShowsSelectionForListTypes(t *testing.T)
```

---

### 3. Git Integration (`internal/git`) - **MEDIUM**

**Current State:** 0% coverage for ~129 lines

**Why it matters:** Git status is displayed in the shell prompt. Incorrect parsing affects user experience.

**Functions to test:**
- `GetStatus` / `GetStatusWithContext` - Main status parsing
- `parseInt` - Helper for parsing ahead/behind counts
- Parsing of various git status formats (staged, unstaged, conflicts, ahead/behind)

**Recommended tests:**
```go
// status_test.go
func TestGetStatus_ParsesBranchName(t *testing.T)
func TestGetStatus_CountsStagedFiles(t *testing.T)
func TestGetStatus_CountsUnstagedFiles(t *testing.T)
func TestGetStatus_DetectsConflicts(t *testing.T)
func TestGetStatus_ParsesAheadBehind(t *testing.T)
func TestGetStatus_ReturnsNilOutsideRepo(t *testing.T)
func TestParseInt_ParsesPositiveNumbers(t *testing.T)
func TestParseInt_IgnoresNonDigits(t *testing.T)
```

**Testing strategy:** Create temporary git repositories with various states for integration tests.

---

### 4. System Resources (`internal/system`) - **MEDIUM**

**Current State:** 0% coverage for ~200 lines across 5 files

**Why it matters:** Platform-specific code (darwin, linux, windows) is notoriously tricky to get right.

**Testing challenges:**
- Platform-specific implementations (`resources_darwin.go`, `resources_linux.go`, `resources_windows.go`)
- Requires mocking system calls or using build tags

**Recommended approach:**
```go
// resources_test.go
func TestGetResources_ReturnsNonNilOnCurrentPlatform(t *testing.T)
func TestGetResources_CPUPercentInValidRange(t *testing.T)
func TestGetResources_RAMValuesAreReasonable(t *testing.T)
func TestGetResources_TimestampIsRecent(t *testing.T)
```

---

### 5. Utils LLM Client (`internal/utils`) - **MEDIUM**

**Current State:** 22.5% coverage - missing tests for `GetLLMClient` and `HideHomeDirPath`

**Missing tests:**
```go
// utils_test.go
func TestHideHomeDirPath_ReplacesHomeWithTilde(t *testing.T)
func TestHideHomeDirPath_LeavesNonHomePathsUnchanged(t *testing.T)
func TestGenerateJsonSchema_PanicsOnError(t *testing.T)

// llmclient_test.go
func TestGetLLMClient_ReturnsClientForFastModel(t *testing.T)
func TestGetLLMClient_ReturnsClientForSlowModel(t *testing.T)
func TestGetLLMClient_UsesProviderFromEnv(t *testing.T)
```

---

## Medium-Priority Improvements

### 6. Idle Summary Generator (`internal/idle`)

**Current State:** 0% coverage

**Key function:** `GenerateSummary` - LLM-based summary of recent commands

**Recommended tests:**
```go
func TestGenerateSummary_ReturnsEmptyWhenNoRecentCommands(t *testing.T)
func TestGenerateSummary_FormatsCommandListCorrectly(t *testing.T)
func TestGenerateSummary_TrimsQuotesFromResponse(t *testing.T)
```

---

### 7. Debounce Package (`pkg/debounce`)

**Current State:** 47.1% coverage

**Missing edge cases:**
- Rapid successive calls
- Cancellation behavior
- Timeout handling

---

## Low-Priority Improvements

### 8. Filesystem Wrapper (`internal/filesystem`)

Simple wrapper around `os` package. Low priority but easy to add:

```go
func TestDefaultFileSystem_ReadFile_ReturnsContents(t *testing.T)
func TestDefaultFileSystem_WriteFile_WritesContents(t *testing.T)
func TestDefaultFileSystem_Open_ReturnsErrorForMissingFile(t *testing.T)
```

### 9. Styles/Terminal Features

These are mostly UI/presentation layer code that's hard to unit test meaningfully.

---

## Testing Infrastructure Recommendations

### 1. Add Mock Interfaces for LLM Clients

Create a `LLMClient` interface to allow dependency injection and testing without real API calls:

```go
type LLMClient interface {
    CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
}
```

### 2. Add Test Fixtures for Git Status

Create `testdata/` directories with sample git status outputs:
```
internal/git/testdata/
├── clean_repo.txt
├── staged_files.txt
├── conflicts.txt
└── ahead_behind.txt
```

### 3. Consider Table-Driven Test Expansion

Many existing tests use table-driven patterns. Extend these for better edge case coverage:

```go
func TestParseInt(t *testing.T) {
    tests := []struct {
        input    string
        expected int
    }{
        {"0", 0},
        {"42", 42},
        {"+5", 5},
        {"-3", 3},  // Note: parseInt ignores non-digits
        {"abc", 0},
        {"12abc34", 1234},
    }
    // ...
}
```

### 4. Add Integration Test Build Tag

For tests that require external services or real system state:

```go
//go:build integration

package git

func TestGetStatus_RealRepository(t *testing.T) {
    // Test against actual git repo
}
```

---

## Suggested Priority Order

1. **Week 1:** `internal/predict` - Core AI functionality
2. **Week 2:** `internal/config` - User settings persistence
3. **Week 3:** `internal/git` - Prompt display correctness
4. **Week 4:** `internal/utils` (remaining), `internal/system`
5. **Ongoing:** `internal/idle`, `pkg/debounce` edge cases

---

## Metrics to Track

| Metric | Current | Target |
|--------|---------|--------|
| Packages with 0% coverage | 9 | 0 |
| Overall package coverage | 67.9% | 90%+ |
| Lines with tests | ~40% | 70%+ |
| Integration tests | 5 files | 10+ files |

---

## Conclusion

The codebase has solid test coverage in core areas like bash parsing, completion, and agent tools. The main gaps are in:

1. **AI/LLM integration** (`predict`, `idle`) - needs mock-based testing
2. **Configuration persistence** (`config`) - needs unit tests for state machine
3. **Platform integration** (`git`, `system`) - needs fixture-based testing

Addressing the HIGH priority items first would significantly improve confidence in the application's core AI features and user configuration handling.
