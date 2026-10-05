package helm

import (
	"errors"
	"fmt"
	"time"

	"go-mini-projects/config"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/storage/driver"
)

type Client struct {
	ActionConfig *action.Configuration
	Namespace    string
}

func NewClient(kubecontext, namespace string) (*Client, error) {
	envSettings := cli.New()
	envSettings.KubeConfig = config.GetKubeConfigPath()
	envSettings.KubeContext = kubecontext

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(envSettings.RESTClientGetter(), namespace, "secret", func(f string, v ...interface{}) {}); err != nil {
		return nil, err
	}

	return &Client{
		ActionConfig: actionConfig,
		Namespace:    namespace,
	}, nil
}

func (h *Client) DeployOrUpgrade(releaseName, chartPath string, values map[string]interface{}) error {
	chart, err := loader.Load(chartPath)
	if err != nil {
		return fmt.Errorf("chart load error: %w", err)
	}

	histClient := action.NewHistory(h.ActionConfig)
	histClient.Max = 1
	_, err = histClient.Run(releaseName)

	if errors.Is(err, driver.ErrReleaseNotFound) {
		fmt.Printf("Release '%s' not found. Performing fresh INSTALL...\n", releaseName)
		installClient := action.NewInstall(h.ActionConfig)
		installClient.ReleaseName = releaseName
		installClient.Namespace = h.Namespace
		installClient.Timeout = 2 * time.Minute
		installClient.Wait = false

		rel, err := installClient.Run(chart, values)
		if err != nil {
			return err
		}
		fmt.Printf("Successfully INSTALLED '%s' (Revision %d)\n", rel.Name, rel.Version)
	} else if err == nil {
		fmt.Printf("Release '%s' found. Performing UPGRADE...\n", releaseName)
		upgradeClient := action.NewUpgrade(h.ActionConfig)
		upgradeClient.Namespace = h.Namespace
		upgradeClient.Timeout = 2 * time.Minute
		upgradeClient.Wait = false

		rel, err := upgradeClient.Run(releaseName, chart, values)
		if err != nil {
			return err
		}
		fmt.Printf("Successfully UPGRADED '%s' to Revision %d\n", rel.Name, rel.Version)
	}

	return nil
}
