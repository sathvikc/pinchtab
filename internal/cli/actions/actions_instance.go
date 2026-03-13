package actions

import (
	"fmt"
	"github.com/pinchtab/pinchtab/internal/cli"
	"github.com/pinchtab/pinchtab/internal/cli/apiclient"
	"github.com/spf13/cobra"
	"net/http"
)

func InstanceStart(client *http.Client, base, token string, cmd *cobra.Command) {
	body := map[string]any{}
	if v, _ := cmd.Flags().GetString("profileId"); v != "" {
		body["profileId"] = v
	}
	if v, _ := cmd.Flags().GetString("mode"); v != "" {
		body["mode"] = v
	}
	if v, _ := cmd.Flags().GetString("port"); v != "" {
		body["port"] = v
	}
	apiclient.DoPost(client, base, token, "/instances/start", body)
}

func InstanceNavigate(client *http.Client, base, token string, args []string) {
	if len(args) < 2 {
		cli.Fatal("Usage: pinchtab instance navigate <instance-id> <url>")
	}

	instID := args[0]
	targetURL := args[1]

	// Instance navigate now works via tab-scoped navigation:
	// open a tab on the instance, then navigate that tab.
	openResp := apiclient.DoPost(client, base, token, fmt.Sprintf("/instances/%s/tabs/open", instID), map[string]any{
		"url": "about:blank",
	})
	tabID, _ := openResp["tabId"].(string)
	if tabID == "" {
		cli.Fatal("failed to open tab for instance %s", instID)
	}

	// apiclient.DoPost auto-prints JSON response.
	apiclient.DoPost(client, base, token, fmt.Sprintf("/tabs/%s/navigate", tabID), map[string]any{
		"url": targetURL,
	})
}

func InstanceLogs(client *http.Client, base, token string, args []string) {
	var instID string

	if len(args) == 0 {
		cli.Fatal("Usage: pinchtab instance logs <instance-id> OR pinchtab instance logs --id <instance-id>")
	}

	if args[0] == "--id" {
		if len(args) < 2 {
			cli.Fatal("Usage: --id requires instance ID")
		}
		instID = args[1]
	} else {
		instID = args[0]
	}

	logs := apiclient.DoGetRaw(client, base, token, fmt.Sprintf("/instances/%s/logs", instID), nil)
	fmt.Println(string(logs))
}

func InstanceStop(client *http.Client, base, token string, args []string) {
	var instID string

	if len(args) == 0 {
		cli.Fatal("Usage: pinchtab instance stop <instance-id> OR pinchtab instance stop --id <instance-id>")
	}

	if args[0] == "--id" {
		if len(args) < 2 {
			cli.Fatal("Usage: --id requires instance ID")
		}
		instID = args[1]
	} else {
		instID = args[0]
	}

	apiclient.DoPost(client, base, token, fmt.Sprintf("/instances/%s/stop", instID), nil)
}
