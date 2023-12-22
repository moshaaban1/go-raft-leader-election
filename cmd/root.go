package cmd

import (
	"fmt"
	"os"
	"os/user"

	"github.com/spf13/cobra"
	"golang.org/x/exp/slog"
	"shaaban.com/raft-leader-election/internal/utils"
	"shaaban.com/raft-leader-election/server"
)

var cmd = &cobra.Command{
	Use:   "server",
	Short: "server runs demo raft leader election protocol",
}

func init() {
	var (
		u, _           = user.Current()
		cfgFile string = u.HomeDir + "/.config.yaml"
		cfg     *server.Config
	)

	cobra.OnInitialize(func() {
		cfg = utils.LoadConfig(cfgFile)
		utils.Logger()
	})

	cmd.Run = func(cmd *cobra.Command, args []string) {
		runServer(cfg)
	}

	cmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", cfgFile, "app config file (default: $HOME/.config.yaml)")
}

func runServer(cfg *server.Config) {
	server := server.New(cfg)

	slog.Info("server running with configuration", "cfg", cfg)

	if err := server.Start(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func Execute() {
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
