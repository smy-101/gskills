package cmd

import (
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/smy-101/gskills/internal/add"
	"github.com/spf13/cobra"
)

const (
	dateFormat   = "2006-01-02 15:04"
	colName      = "Name"
	colUpdatedAt = "Updated At"
	colSourceURL = "Source URL"
	emptyMsg     = "No skills installed yet."
	usageHint    = "Use 'gskills add <url>' to install a skill."
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有已安装的技能",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeList()
	},
}

// executeList loads the registry and displays a table of all installed skills.
func executeList() error {
	skills, err := add.LoadRegistry()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	if len(skills) == 0 {
		fmt.Println(emptyMsg)
		fmt.Println(usageHint)
		return nil
	}

	cnf := tablewriter.Config{
		Header: tw.CellConfig{
			Alignment: tw.CellAlignment{Global: tw.AlignCenter},
		},
		Row: tw.CellConfig{
			Alignment: tw.CellAlignment{Global: tw.AlignLeft},
		},
	}

	table := tablewriter.NewTable(os.Stdout, tablewriter.WithConfig(cnf))
	table.Header(colName, colUpdatedAt, colSourceURL)

	for _, skill := range skills {
		updatedAt := skill.UpdatedAt.Format(dateFormat)
		table.Append(skill.Name, updatedAt, skill.SourceURL)
	}

	if err := table.Render(); err != nil {
		return fmt.Errorf("failed to render table: %w", err)
	}

	fmt.Printf("\nTotal: %d skills\n", len(skills))

	return nil
}
