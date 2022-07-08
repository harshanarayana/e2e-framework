package framework

import (
	"fmt"
	"sigs.k8s.io/e2e-framework/pkg/framework/types"
)

var providerRegistry map[string]types.ClusterProviderGenerator

func RegisterProvider(providerName string, f types.ClusterProviderGenerator) {
	if _, ok := providerRegistry[providerName]; ok {
		panic(fmt.Sprintf("a provider with name %s is already registered. Duplicate registering is not allowed", providerName))
	}
	providerRegistry[providerName] = f
}

func GetProviderGenerator(providerName string) types.ClusterProviderGenerator {
	if f, ok := providerRegistry[providerName]; !ok {
		panic(fmt.Sprintf("no provider with name %s is registered", providerName))
	} else {
		return f
	}
}

func WithKubernetesVersion(version string) types.CreateOptions {
	return func(config *types.ClusterConfig) {
		config.K8SVersion = version
	}
}

func WithInitConfig(initConfig string) types.CreateOptions {
	return func(config *types.ClusterConfig) {
		config.InitConfig = initConfig
	}
}

func WithArgs(args ...string) types.CreateOptions {
	return func(config *types.ClusterConfig) {
		config.Args = append(config.Args, args...)
	}
}

func WithName(name string) types.CreateOptions {
	return func(config *types.ClusterConfig) {
		config.Name = name
	}
}

func init() {
	providerRegistry = make(map[string]types.ClusterProviderGenerator, 0)
}
