// Package rbridge provides Go-R integration for statistical analysis.
package rbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/guidiju-50/pandora/ANALYSIS/internal/config"
	"go.uber.org/zap"
)

// Executor handles execution of R scripts from Go.
type Executor struct {
	config config.RConfig
	logger *zap.Logger
}

// NewExecutor creates a new R executor.
func NewExecutor(cfg config.RConfig, logger *zap.Logger) *Executor {
	return &Executor{
		config: cfg,
		logger: logger,
	}
}

// ExecuteOptions holds options for R script execution.
type ExecuteOptions struct {
	Script     string                 // Script name (without path)
	Args       map[string]interface{} // Arguments to pass to R
	OutputFile string                 // Expected output file
	WorkDir    string                 // Working directory
}

// Result holds the result of R script execution.
type Result struct {
	Success    bool                   `json:"success"`
	Output     string                 `json:"output"`
	Error      string                 `json:"error,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"`
	OutputFile string                 `json:"output_file,omitempty"`
}

// Execute runs an R script with the given arguments.
func (e *Executor) Execute(ctx context.Context, opts ExecuteOptions) (*Result, error) {
	scriptPath := filepath.Join(e.config.ScriptsPath, opts.Script)

	// Verify script exists
	if _, err := os.Stat(scriptPath); err != nil {
		return nil, fmt.Errorf("script not found: %s", scriptPath)
	}

	// Create args file
	argsFile, err := e.createArgsFile(opts.Args, opts.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("creating args file: %w", err)
	}
	defer os.Remove(argsFile)

	// Create output file path
	outputFile := opts.OutputFile
	if outputFile == "" {
		outputFile = filepath.Join(opts.WorkDir, "r_output.json")
	}

	e.logger.Info("executing R script",
		zap.String("script", opts.Script),
		zap.String("args_file", argsFile),
	)

	// Build command
	cmd := exec.CommandContext(ctx, e.config.Path, scriptPath, argsFile, outputFile)
	cmd.Dir = opts.WorkDir

	// Set R library path
	if e.config.LibsPath != "" {
		cmd.Env = append(os.Environ(), "R_LIBS_USER="+e.config.LibsPath)
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute
	err = cmd.Run()

	result := &Result{
		Success:    err == nil,
		Output:     stdout.String(),
		OutputFile: outputFile,
	}

	if err != nil {
		result.Error = stderr.String()
		e.logger.Error("R script failed",
			zap.String("script", opts.Script),
			zap.String("stderr", stderr.String()),
			zap.Error(err),
		)
		return result, fmt.Errorf("R script failed: %w", err)
	}

	// Parse output file if it exists
	if _, err := os.Stat(outputFile); err == nil {
		data, err := os.ReadFile(outputFile)
		if err == nil {
			var output map[string]interface{}
			if err := json.Unmarshal(data, &output); err == nil {
				result.Data = output
			}
		}
	}

	e.logger.Info("R script completed",
		zap.String("script", opts.Script),
	)

	return result, nil
}

// createArgsFile creates a JSON file with arguments for R.
func (e *Executor) createArgsFile(args map[string]interface{}, workDir string) (string, error) {
	argsFile := filepath.Join(workDir, "r_args.json")

	data, err := json.MarshalIndent(args, "", "  ")
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(argsFile, data, 0644); err != nil {
		return "", err
	}

	return argsFile, nil
}

// ExecuteTemplate executes an R script from a template.
func (e *Executor) ExecuteTemplate(ctx context.Context, templateName string, data interface{}, workDir string) (*Result, error) {
	// Load template
	templatePath := filepath.Join(e.config.ScriptsPath, "templates", templateName)
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	// Generate script
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	// Write temporary script
	scriptFile := filepath.Join(workDir, "generated_script.R")
	if err := os.WriteFile(scriptFile, buf.Bytes(), 0644); err != nil {
		return nil, fmt.Errorf("writing script: %w", err)
	}
	defer os.Remove(scriptFile)

	// Execute
	return e.ExecuteScript(ctx, scriptFile, workDir)
}

// ExecuteScript executes an R script file directly.
func (e *Executor) ExecuteScript(ctx context.Context, scriptPath, workDir string) (*Result, error) {
	e.logger.Info("executing R script directly",
		zap.String("script", scriptPath),
	)

	cmd := exec.CommandContext(ctx, e.config.Path, scriptPath)
	cmd.Dir = workDir

	if e.config.LibsPath != "" {
		cmd.Env = append(os.Environ(), "R_LIBS_USER="+e.config.LibsPath)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &Result{
		Success: err == nil,
		Output:  stdout.String(),
	}

	if err != nil {
		result.Error = stderr.String()
		return result, fmt.Errorf("R script failed: %w", err)
	}

	return result, nil
}

// InstallPackages installs R packages.
func (e *Executor) InstallPackages(ctx context.Context, packages []string) error {
	e.logger.Info("installing R packages", zap.Strings("packages", packages))

	script := `
packages <- c(%s)
for (pkg in packages) {
    if (!require(pkg, character.only = TRUE)) {
        install.packages(pkg, repos = "https://cran.r-project.org")
    }
}
`

	// Format package list
	pkgList := ""
	for i, pkg := range packages {
		if i > 0 {
			pkgList += ", "
		}
		pkgList += fmt.Sprintf(`"%s"`, pkg)
	}

	finalScript := fmt.Sprintf(script, pkgList)

	cmd := exec.CommandContext(ctx, e.config.Path, "-e", finalScript)
	output, err := cmd.CombinedOutput()
	if err != nil {
		e.logger.Error("failed to install packages",
			zap.String("output", string(output)),
			zap.Error(err),
		)
		return fmt.Errorf("installing packages: %w", err)
	}

	return nil
}

// CheckPackages checks if required R packages are installed.
func (e *Executor) CheckPackages(ctx context.Context, packages []string) (map[string]bool, error) {
	result := make(map[string]bool)

	for _, pkg := range packages {
		script := fmt.Sprintf(`cat(require("%s", quietly = TRUE))`, pkg)
		cmd := exec.CommandContext(ctx, e.config.Path, "-e", script)
		output, err := cmd.Output()
		if err != nil {
			result[pkg] = false
		} else {
			result[pkg] = string(output) == "TRUE"
		}
	}

	return result, nil
}
