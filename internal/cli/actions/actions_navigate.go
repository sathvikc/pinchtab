package actions

import (
	"fmt"
	"net/http"

	"github.com/pinchtab/pinchtab/internal/cli/apiclient"
	"github.com/pinchtab/pinchtab/internal/cli/output"
	"github.com/spf13/cobra"
)

// Back navigates the current (or specified) tab back in history.
func Back(client *http.Client, base, token string, cmd *cobra.Command) {
	tabID, _ := cmd.Flags().GetString("tab")
	path := "/back"
	if tabID != "" {
		path = "/tabs/" + tabID + "/back"
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		apiclient.DoPost(client, base, token, path, nil)
		return
	}

	result := apiclient.DoPostQuiet(client, base, token, path, nil)
	if url, ok := result["url"].(string); ok {
		output.Value(url)
	} else {
		output.Success()
	}

	snap, _ := cmd.Flags().GetBool("snap")
	snapDiff, _ := cmd.Flags().GetBool("snap-diff")
	if snap || snapDiff {
		fetchAndPrintSnapshot(client, base, token, tabID, snapDiff)
	}
	text, _ := cmd.Flags().GetBool("text")
	if text {
		fetchAndPrintText(client, base, token, tabID)
	}
}

// Forward navigates the current (or specified) tab forward in history.
func Forward(client *http.Client, base, token string, cmd *cobra.Command) {
	tabID, _ := cmd.Flags().GetString("tab")
	path := "/forward"
	if tabID != "" {
		path = "/tabs/" + tabID + "/forward"
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		apiclient.DoPost(client, base, token, path, nil)
		return
	}

	result := apiclient.DoPostQuiet(client, base, token, path, nil)
	if url, ok := result["url"].(string); ok {
		output.Value(url)
	} else {
		output.Success()
	}

	snap, _ := cmd.Flags().GetBool("snap")
	snapDiff, _ := cmd.Flags().GetBool("snap-diff")
	if snap || snapDiff {
		fetchAndPrintSnapshot(client, base, token, tabID, snapDiff)
	}
	text, _ := cmd.Flags().GetBool("text")
	if text {
		fetchAndPrintText(client, base, token, tabID)
	}
}

// Reload reloads the current (or specified) tab.
func Reload(client *http.Client, base, token string, cmd *cobra.Command) {
	tabID, _ := cmd.Flags().GetString("tab")
	path := "/reload"
	if tabID != "" {
		path = "/tabs/" + tabID + "/reload"
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		apiclient.DoPost(client, base, token, path, nil)
		return
	}

	apiclient.DoPostQuiet(client, base, token, path, nil)
	output.Success()

	snap, _ := cmd.Flags().GetBool("snap")
	snapDiff, _ := cmd.Flags().GetBool("snap-diff")
	if snap || snapDiff {
		fetchAndPrintSnapshot(client, base, token, tabID, snapDiff)
	}
	text, _ := cmd.Flags().GetBool("text")
	if text {
		fetchAndPrintText(client, base, token, tabID)
	}
}

func Navigate(client *http.Client, base, token string, url string, cmd *cobra.Command) string {
	path, body := buildNavigateRequest(client, base, token, url, cmd)

	// JSON output mode
	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		result := apiclient.DoPost(client, base, token, path, body)
		apiclient.SuggestNextAction("navigate", result)
		return tabIDFromNavigateResult(result)
	}

	resultTabID := postNavigateQuiet(client, base, token, path, body)
	if resultTabID != "" {
		fmt.Println(resultTabID)
	}

	// If --snap or --snap-diff flag is set, fetch and output snapshot
	snap, _ := cmd.Flags().GetBool("snap")
	snapDiff, _ := cmd.Flags().GetBool("snap-diff")
	if snap || snapDiff {
		fetchAndPrintSnapshot(client, base, token, resultTabID, snapDiff)
	}

	return resultTabID
}

func buildNavigateRequest(client *http.Client, base, token, url string, cmd *cobra.Command) (string, map[string]any) {
	body := map[string]any{"url": url}
	newTab, _ := cmd.Flags().GetBool("new-tab")
	if newTab {
		body["newTab"] = true
	}
	if v, _ := cmd.Flags().GetBool("block-images"); v {
		body["blockImages"] = true
	}
	if v, _ := cmd.Flags().GetBool("block-ads"); v {
		body["blockAds"] = true
	}
	tabID, _ := cmd.Flags().GetString("tab")
	path := "/navigate"
	explicitTab := cmd.Flags().Changed("tab")
	// Don't use tab-specific path when creating a new tab. If the tab came from
	// the saved current-tab state file and no longer exists, treat it as no
	// current tab and let /navigate create one.
	if tabID != "" && !newTab && (explicitTab || tabExists(client, base, token, tabID)) {
		path = "/tabs/" + tabID + "/navigate"
	}

	return path, body
}

func postNavigateQuiet(client *http.Client, base, token, path string, body map[string]any) string {
	result := apiclient.DoPostQuiet(client, base, token, path, body)
	return tabIDFromNavigateResult(result)
}

func tabIDFromNavigateResult(result map[string]any) string {
	if tid, ok := result["tabId"].(string); ok && tid != "" {
		return tid
	}
	return ""
}
