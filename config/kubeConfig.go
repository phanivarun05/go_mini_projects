package config

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func GetKubeConfigPath() string {
	if env := os.Getenv("KUBECONFIG"); env != "" {
		return env
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kube", "config")
}

func LoadRawKubeConfig() (*api.Config, error) {
	return clientcmd.LoadFromFile(GetKubeConfigPath())
}
