package main

//ui#13
import (
	"fmt"
	"os"

	clientv2helpers "cosmossdk.io/client/v2/helpers"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	sdkversion "github.com/cosmos/cosmos-sdk/version"

	"github.com/qorechain/qorechain-core/app"
	"github.com/qorechain/qorechain-core/cmd/qorechaind/cmd"
)

// Code-level default for the binary name so node_info's
// application_version.app_name reports "qorechaind" instead of the Cosmos SDK
// placeholder "<appd>" even when the binary is built without version ldflags.
// Build-time ldflags (Makefile) still set Version/Commit; those are untouched.
func init() {
	sdkversion.AppName = "qorechaind"
}

func main() {
	rootCmd := cmd.NewRootCmd()
	if err := svrcmd.Execute(rootCmd, clientv2helpers.EnvPrefix, app.DefaultNodeHome); err != nil {
		fmt.Fprintln(rootCmd.OutOrStderr(), err)
		os.Exit(1)
	}
}
