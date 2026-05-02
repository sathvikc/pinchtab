package main

import (
	browseractions "github.com/pinchtab/pinchtab/internal/cli/actions"
	"github.com/spf13/cobra"
)

var cookiesCmd = &cobra.Command{
	Use:   "cookies",
	Short: "Manage browser cookies",
	Long:  "Commands for managing browser cookies.",
}

var cookiesClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all browser cookies",
	Long:  "Clear all browser cookies via CDP. This affects all origins.",
	Run: func(cmd *cobra.Command, args []string) {
		runCLI(func(rt cliRuntime) {
			browseractions.CookiesClear(rt.client, rt.base, rt.token, cmd)
		})
	},
}

func init() {
	cookiesCmd.AddCommand(cookiesClearCmd)
}
