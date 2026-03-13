package actions

import (
	"fmt"
	"github.com/pinchtab/pinchtab/internal/cli"
	"github.com/pinchtab/pinchtab/internal/cli/apiclient"
	"net/http"
	"os"
)

// TabList lists all open tabs.
func TabList(client *http.Client, base, token string) {
	apiclient.DoGet(client, base, token, "/tabs", nil)
}

// TabNew opens a new tab (exported for cobra subcommand).
func TabNew(client *http.Client, base, token string, body map[string]any) {
	// Check if any instances are running
	instances := getInstances(client, base, token)
	if len(instances) == 0 {
		fmt.Fprintln(os.Stderr, cli.StyleStderr(cli.WarningStyle, "No instances running, launching default..."))
		launchInstance(client, base, token, "default")
		fmt.Fprintln(os.Stderr, cli.StyleStderr(cli.SuccessStyle, "Instance launched"))
	}
	apiclient.DoPost(client, base, token, "/tab", body)
}

// TabClose closes a tab by ID (exported for cobra subcommand).
func TabClose(client *http.Client, base, token string, tabID string) {
	apiclient.DoPost(client, base, token, "/tab", map[string]any{
		"action": "close",
		"tabId":  tabID,
	})
}
