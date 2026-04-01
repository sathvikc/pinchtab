package main

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"time"

	"github.com/pinchtab/pinchtab/internal/cli"
	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Open the dashboard in your browser",
	Long:  "Resolve the dashboard URL, copy the auth token to clipboard, and open the browser.",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := loadConfig()
		noOpen, _ := cmd.Flags().GetBool("no-open")
		portOverride, _ := cmd.Flags().GetString("port")
		runDashboardCommand(cfg.Port, cfg.Bind, cfg.Token, noOpen, portOverride)
	},
}

func init() {
	dashboardCmd.GroupID = "primary"
	dashboardCmd.Flags().Bool("no-open", false, "Print URL without opening the browser")
	dashboardCmd.Flags().String("port", "", "Override dashboard port")
	rootCmd.AddCommand(dashboardCmd)
}

func runDashboardCommand(cfgPort, cfgBind, cfgToken string, noOpen bool, portOverride string) {
	port := resolvePort(cfgPort, portOverride)
	host := resolveHost(cfgBind)
	url := fmt.Sprintf("http://%s:%s", host, port)

	// Health check
	if !isDashboardReachable(host, port) {
		fmt.Println(cli.StyleStdout(cli.WarningStyle, "  Dashboard doesn't appear to be running."))
		fmt.Printf("  Start with %s or %s\n\n",
			cli.StyleStdout(cli.CommandStyle, "pinchtab server"),
			cli.StyleStdout(cli.CommandStyle, "pinchtab daemon start"))
	}

	fmt.Printf("%s %s\n", cli.StyleStdout(cli.HeadingStyle, "Dashboard:"), cli.StyleStdout(cli.ValueStyle, url))

	// Copy token to clipboard (never embed in URL)
	if cfgToken != "" {
		if err := copyToClipboard(cfgToken); err == nil {
			fmt.Println(cli.StyleStdout(cli.SuccessStyle, "  Token copied to clipboard") +
				cli.StyleStdout(cli.MutedStyle, " — paste it on the login page."))
		} else {
			fmt.Println(cli.StyleStdout(cli.WarningStyle, "  Token could not be copied to clipboard."))
		}
	}

	// Open browser
	if !noOpen {
		if err := openBrowser(url); err == nil {
			fmt.Println(cli.StyleStdout(cli.SuccessStyle, "  Opened in your browser."))
		} else {
			fmt.Printf("  Open this URL in your browser: %s\n", cli.StyleStdout(cli.ValueStyle, url))
		}
	} else {
		fmt.Printf("  Open this URL in your browser: %s\n", cli.StyleStdout(cli.ValueStyle, url))
	}
}

func resolvePort(cfgPort, override string) string {
	if override != "" {
		return override
	}
	if cfgPort != "" {
		return cfgPort
	}
	return "9870"
}

func resolveHost(bind string) string {
	switch bind {
	case "", "loopback", "localhost", "lan", "0.0.0.0":
		return "127.0.0.1"
	default:
		return bind
	}
}

func isDashboardReachable(host, port string) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 1*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
