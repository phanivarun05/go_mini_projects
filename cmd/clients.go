package cmd

import (
	deploymentmap "go-mini-projects/internal/deployment-map"
	"go-mini-projects/internal/helm"
	"go-mini-projects/internal/k8s"

	"github.com/spf13/cobra"
)

var (
	flagContext   string
	flagNamespace string
)

var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect pods and group them by deployment lineage (ArgoCD / Helm / Standalone)",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Resolve context (use flag if provided, else prompt interactively)
		targetCtx, err := k8s.SelectContextOrPrompt(flagContext)
		if err != nil {
			return err
		}

		// 2. Initialize K8s client for resolved context
		k8sClient, err := k8s.NewClient(targetCtx)
		if err != nil {
			return err
		}

		// 3. Resolve namespace (use flag if provided, else list and prompt)
		targetNs, err := k8sClient.SelectNamespaceOrPrompt(flagNamespace)
		if err != nil {
			return err
		}

		// 4. Run classification engine
		return deploymentmap.InspectLineage(k8sClient, targetNs)
	},
}

var deployCmd = &cobra.Command{
	Use:   "deploy [RELEASE_NAME] [CHART_PATH]",
	Short: "Install or upgrade a Helm chart",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		releaseName := args[0]
		chartPath := args[1]

		targetCtx, err := k8s.SelectContextOrPrompt(flagContext)
		if err != nil {
			return err
		}

		k8sClient, err := k8s.NewClient(targetCtx)
		if err != nil {
			return err
		}

		targetNs, err := k8sClient.SelectNamespaceOrPrompt(flagNamespace)
		if err != nil {
			return err
		}

		helmClient, err := helm.NewClient(targetCtx, targetNs)
		if err != nil {
			return err
		}

		values := map[string]interface{}{
			"replicaCount": 2,
			"image": map[string]interface{}{
				"repository": "nginx",
				"tag":        "alpine",
			},
		}

		return helmClient.DeployOrUpgrade(releaseName, chartPath, values)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&flagContext, "context", "c", "", "Target Kubernetes context/cluster")
	rootCmd.PersistentFlags().StringVarP(&flagNamespace, "namespace", "n", "", "Target Kubernetes namespace")

	rootCmd.AddCommand(inspectCmd)
	rootCmd.AddCommand(deployCmd)
}
