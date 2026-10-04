package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "k8s-Helm-client",
	Short: "A CLI tool to interact with Kubernetes clusters and Helm charts",
	Long: `k8s-Helm-client is a command-line tool that allows users to interact with Kubernetes clusters and manage Helm charts. 
	It provides functionalities to list pods, deploy applications, and manage Helm releases.`,
}

func Execute() error {
	return rootCmd.Execute()
}
