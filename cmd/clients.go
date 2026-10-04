package cmd

import (
	"go-mini-projects/internal/k8s"

	"github.com/spf13/cobra"
)

func runK8s(cmd *cobra.Command, args []string) {
	k8s.K8sClient()
}

/*func runHelm(cmd *cobra.Command, args []string) {}

func runDeploy(cmd *cobra.Command, args []string) {}
*/

var k8sCmd = &cobra.Command{
	Use:   "K8s",
	Short: "Interact with Kubernetes clusters",
	Run:   runK8s,
}

/*var helmcmd = &cobra.Command{
	Use:   "Helm",
	Short: "Manage Helm charts",
}

var deployCmd = &cobra.Command{
	Use:   "Deploy",
	Short: "Deploy an application to the Kubernetes cluster",
}
*/

func init() {
	rootCmd.AddCommand(k8sCmd)
}
