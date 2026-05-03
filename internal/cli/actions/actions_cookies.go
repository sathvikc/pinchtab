package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/pinchtab/pinchtab/internal/cli/apiclient"
	"github.com/spf13/cobra"
)

// CookiesClear clears all browser cookies.
func CookiesClear(client *http.Client, base, token string, cmd *cobra.Command) {
	result := apiclient.DoDelete(client, base, token, "/cookies", nil)
	if result == nil {
		fmt.Fprintln(os.Stderr, "ERROR: cookies: clear failed")
		os.Exit(2)
	}

	jsonOut, _ := cmd.Flags().GetBool("json")
	if jsonOut {
		out, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(out))
	} else {
		fmt.Println("OK")
	}
}
