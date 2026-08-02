package main

import (
	"context"
	"fmt"

	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/spf13/cobra"
)

func newClassifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "classify",
		Short: "Developer tools for money classification (not an end-user path)",
	}
	cmd.AddCommand(newClassifyCategoriesCmd())
	return cmd
}

func newClassifyCategoriesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "categories",
		Short: "List system category taxonomy",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pool, cleanup, err := openImportDB()
			if err != nil {
				return err
			}
			defer cleanup()

			categories, err := repository.NewCategories(pool).List(context.Background())
			if err != nil {
				return err
			}

			if len(categories) == 0 {
				fmt.Println("No categories found (run migrate up)")
				return nil
			}

			fmt.Println("Categories")
			for _, cat := range categories {
				system := ""
				if cat.IsSystem {
					system = " system"
				}
				fmt.Printf("  %-18s  %-20s  kind=%-8s%s\n",
					cat.Slug,
					cat.DisplayName,
					cat.Kind,
					system,
				)
			}
			return nil
		},
	}
}
