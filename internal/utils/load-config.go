package utils

import (
	"log"

	"github.com/spf13/viper"
	"shaaban.com/raft-leader-election/server"
)

func LoadConfig(cfgFile string) *server.Config {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}

	if err := viper.ReadInConfig(); err != nil {
		log.Fatal(err)
	}

	return &server.Config{
		Port:              viper.GetInt("port"),
		Node:              viper.GetString("node"),
		Peers:             viper.GetStringSlice("peers"),
		HeartBeatInterval: viper.GetDuration("heartbeat_interval"),
		HeartBeatTimeout:  viper.GetDuration("heartbeat_timeout"),
	}
}
