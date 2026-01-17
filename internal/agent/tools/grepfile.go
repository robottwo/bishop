package tools

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/robottwo/bishop/internal/environment"
	"github.com/robottwo/bishop/internal/utils"
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

var GrepFileToolDefinition = openai.Tool{
	Type: "function",
	Function: &openai.FunctionDefinition{
		Name:        "grep_file",
		Description: "Search for a pattern (regex) in a file and return matching lines with line numbers. Similar to grep command.",
		Parameters: utils.GenerateJsonSchema(struct {
			Path         string `json:"path" description:"Absolute path to the file to search" required:"true"`
			Pattern      string `json:"pattern" description:"Regular expression pattern to search for" required:"true"`
			ContextLines int    `json:"context_lines" description:"Optional. Number of lines to show before and after each match (like grep -C). Default is 0." required:"false"`
		}{}),
	},
}

func GrepFileTool(runner *interp.Runner, logger *zap.Logger, params map[string]any) string {
	path, ok := params["path"].(string)
	if !ok {
		logger.Error("The grep_file tool failed to parse parameter 'path'")
		return failedToolResponse("The grep_file tool failed to parse parameter 'path'")
	}

	pattern, ok := params["pattern"].(string)
	if !ok {
		logger.Error("The grep_file tool failed to parse parameter 'pattern'")
		return failedToolResponse("The grep_file tool failed to parse parameter 'pattern'")
	}

	if !filepath.IsAbs(path) {
		path = filepath.Join(environment.GetPwd(runner), path)
	}

	contextLines := 0
	contextLinesVal, contextLinesExists := params["context_lines"]
	if contextLinesExists {
		contextLinesFloat, ok := contextLinesVal.(float64)
		if !ok {
			logger.Error("The grep_file tool failed to parse parameter 'context_lines'")
			return failedToolResponse("The grep_file tool failed to parse parameter 'context_lines'")
		contextLines = int(contextLinesFloat)
		if contextLines < 0 {
			logger.Error("grep_file tool received negative context_lines")
			return failedToolResponse("context_lines must be non-negative")
		}
		contextLines = int(contextLinesFloat)
	}

	// Compile the regex pattern
	re, err := regexp.Compile(pattern)
	if err != nil {
		logger.Error("grep_file tool received invalid regex pattern", zap.Error(err))
		return failedToolResponse(fmt.Sprintf("Invalid regex pattern: %s", err))
	}

	file, err := os.Open(path)
	if err != nil {
		logger.Error("grep_file tool received invalid path", zap.Error(err))
		return failedToolResponse(fmt.Sprintf("Error opening file: %s", err))
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Warn("failed to close file", zap.Error(err))
		}
	}()

	agentName := environment.GetAgentName(runner)
	printToolMessage(fmt.Sprintf("%s: I'm searching the following file:", agentName))
	printToolPath(utils.HideHomeDirPath(runner, path))

	// Read all lines
	var lines []string
	// Check file size before reading
	fileInfo, err := file.Stat()
	if err != nil {
		logger.Error("grep_file tool error getting file info", zap.Error(err))
		return failedToolResponse(fmt.Sprintf("Error reading file info: %s", err))
	}
	
	const maxFileSize = 100 * 1024 * 1024 // 100MB limit
	if fileInfo.Size() > maxFileSize {
		return failedToolResponse(fmt.Sprintf("File too large (%d bytes). Maximum size is %d bytes", fileInfo.Size(), maxFileSize))
	}
	
	// Read all lines
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		logger.Error("grep_file tool error reading file", zap.Error(err))
		return failedToolResponse(fmt.Sprintf("Error reading file: %s", err))
	}

	// Find matches
	matchedLines := make(map[int]bool)
	for i, line := range lines {
		if re.MatchString(line) {
			matchedLines[i] = true
		}
	}

	// If no matches found
	if len(matchedLines) == 0 {
		return "No matches found."
	}

	// Build output with context lines
	var result strings.Builder
	// Build output with context lines
	var result strings.Builder
	
	// Collect and sort all lines to output
	lineNums := make([]int, 0, len(matchedLines)*(2*contextLines+1))
	for lineNum := range matchedLines {
		for i := lineNum - contextLines; i <= lineNum+contextLines; i++ {
			if i >= 0 && i < len(lines) {
				lineNums = append(lineNums, i)
			}
		}
	}
	
	// Remove duplicates and sort
	seen := make(map[int]bool)
	uniqueLines := make([]int, 0, len(lineNums))
	for _, num := range lineNums {
		if !seen[num] {
			seen[num] = true
			uniqueLines = append(uniqueLines, num)
		}
	}
	sort.Ints(uniqueLines)
	
	// Generate output
	previousLine := -2
	for _, i := range uniqueLines {

	// Add matched lines and their context
	for lineNum := range matchedLines {
		for i := lineNum - contextLines; i <= lineNum+contextLines; i++ {
			if i >= 0 && i < len(lines) {
				outputLines[i] = true
			}
		}
	}

	// Convert to sorted output
	previousLine := -2 // Track if we need to add separator
	for i := 0; i < len(lines); i++ {
		if !outputLines[i] {
			continue
		}

		// Add separator if there's a gap in line numbers
		if previousLine >= 0 && i > previousLine+1 {
			result.WriteString("--\n")
		}

		// Format: line_number:line_content
		// Mark matched lines with : and context lines with -
		lineNum := i + 1 // 1-indexed line numbers
		if matchedLines[i] {
			result.WriteString(fmt.Sprintf("%d:%s\n", lineNum, lines[i]))
		} else {
			result.WriteString(fmt.Sprintf("%d-%s\n", lineNum, lines[i]))
		}

		previousLine = i
	}

	output := result.String()

	// Respect MAX_VIEW_SIZE constant for output truncation
	if len(output) > MAX_VIEW_SIZE {
		return output[:MAX_VIEW_SIZE] + "\n<bish:truncated />"
	}

	return output
}
