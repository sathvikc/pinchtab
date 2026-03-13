package actions

import (
	"fmt"
	"github.com/pinchtab/pinchtab/internal/cli/apiclient"
	"github.com/spf13/cobra"
	"net/http"
)

func Navigate(client *http.Client, base, token string, url string, cmd *cobra.Command) {
	body := map[string]any{"url": url}
	if v, _ := cmd.Flags().GetBool("new-tab"); v {
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
	if tabID != "" {
		path = fmt.Sprintf("/tabs/%s/navigate", tabID)
	}
	result := apiclient.DoPost(client, base, token, path, body)
	apiclient.SuggestNextAction("navigate", result)
}
