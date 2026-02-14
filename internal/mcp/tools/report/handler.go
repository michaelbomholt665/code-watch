package report

import (
	"circular/internal/mcp/contracts"
	"context"
)

type Manager interface {
	GenerateMarkdownReport(ctx context.Context, in contracts.ReportGenerateMarkdownInput) (contracts.ReportGenerateMarkdownOutput, error)
}

func HandleGenerateMarkdown(ctx context.Context, mgr Manager, in contracts.ReportGenerateMarkdownInput) (contracts.ReportGenerateMarkdownOutput, error) {
	return mgr.GenerateMarkdownReport(ctx, in)
}
