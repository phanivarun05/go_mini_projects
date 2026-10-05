package k8s

import (
	"context"
	"fmt"
	"time"

	"go-mini-projects/config"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	Clientset *kubernetes.Clientset
	Context   string
	Namespace string
}

// NewClient initializes a Kubernetes clientset for a targeted context
func NewClient(ctxName string) (*Client, error) {
	kubeconfig := config.GetKubeConfigPath()
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = kubeconfig

	configOverrides := &clientcmd.ConfigOverrides{
		CurrentContext: ctxName,
	}

	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides).ClientConfig()
	if err != nil {
		return nil, err
	}

	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &Client{
		Clientset: cs,
		Context:   ctxName,
	}, nil
}

// SelectContextOrPrompt checks if targetContext is set; if empty, lists contexts and prompts user
func SelectContextOrPrompt(targetContext string) (string, error) {
	if targetContext != "" {
		return targetContext, nil
	}

	rawCfg, err := config.LoadRawKubeConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	fmt.Println("\nAvailable Contexts (Clusters):")
	var contexts []string
	for ctxName := range rawCfg.Contexts {
		contexts = append(contexts, ctxName)
		if ctxName == rawCfg.CurrentContext {
			fmt.Printf(" * %s (current default)\n", ctxName)
		} else {
			fmt.Printf("   %s\n", ctxName)
		}
	}

	var selected string
	fmt.Print("\nEnter target context [Press Enter for default]: ")
	fmt.Scanln(&selected)

	if selected == "" {
		return rawCfg.CurrentContext, nil
	}
	return selected, nil
}

// SelectNamespaceOrPrompt checks if targetNamespace is set; if empty, lists namespaces and prompts user
func (c *Client) SelectNamespaceOrPrompt(targetNamespace string) (string, error) {
	if targetNamespace != "" {
		return targetNamespace, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	nsList, err := c.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list namespaces: %w", err)
	}

	fmt.Printf("\nAvailable Namespaces in '%s':\n", c.Context)
	for _, ns := range nsList.Items {
		fmt.Printf(" - %s\n", ns.Name)
	}

	var selected string
	fmt.Print("\nEnter target namespace [default: 'default']: ")
	fmt.Scanln(&selected)

	if selected == "" {
		return "default", nil
	}
	return selected, nil
}
