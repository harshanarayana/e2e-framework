package providers

import (
	"bytes"
	"fmt"
	"github.com/vladimirvivien/gexe"
	"io"
	"io/ioutil"
	log "k8s.io/klog/v2"
	"os"
	"sigs.k8s.io/e2e-framework/pkg/framework"
	"sigs.k8s.io/e2e-framework/pkg/framework/types"
	"strings"
	"sync"
)

type kindCluster struct {
	cfg        *types.ClusterConfig
	executor   *gexe.Echo
	kubeConfig string
	fetchOnce  sync.Once
}

func (k *kindCluster) saveKubeConfig() (err error) {
	kubecfg := fmt.Sprintf("%s-kubecfg", k.cfg.Name)

	p := k.executor.StartProc(fmt.Sprintf(`kind get kubeconfig --name %s`, k.cfg.Name))
	if p.Err() != nil {
		return fmt.Errorf("kind get kubeconfig: %w", p.Err())
	}
	var stdout bytes.Buffer
	if _, err := stdout.ReadFrom(p.StdOut()); err != nil {
		return fmt.Errorf("kind kubeconfig stdout bytes: %w", err)
	}
	if p.Wait().Err() != nil {
		return fmt.Errorf("kind get kubeconfig: %s: %w", p.Result(), p.Err())
	}

	file, err := ioutil.TempFile("", fmt.Sprintf("kind-cluser-%s", kubecfg))
	if err != nil {
		return fmt.Errorf("kind kubeconfig file: %w", err)
	}
	defer file.Close()

	k.kubeConfig = file.Name()

	if n, err := io.Copy(file, &stdout); n == 0 || err != nil {
		return fmt.Errorf("kind kubecfg file: bytes copied: %d: %w]", n, err)
	}
	return nil
}

func (k *kindCluster) clusterExists(name string) (string, bool) {
	clusters := k.executor.Run("kind get clusters")
	for _, c := range strings.Split(clusters, "\n") {
		if c == name {
			return clusters, true
		}
	}
	return clusters, false
}

func (k *kindCluster) installKindIfRequired() error {
	if k.executor.Prog().Avail("kind") == "" {
		log.V(4).Infof(`kind not found, installing with go install sigs.k8s.io/kind@%s`, k.cfg.K8SVersion)
		if err := k.installKind(); err != nil {
			return err
		}
	}
	return nil
}

func (k *kindCluster) installKind() error {
	log.V(4).Infof("Installing: go install sigs.k8s.io/kind@%s", k.cfg.K8SVersion)
	p := k.executor.RunProc(fmt.Sprintf("go install sigs.k8s.io/kind@%s", k.cfg.K8SVersion))
	if p.Err() != nil {
		return fmt.Errorf("failed to install kind: %s", p.Err())
	}

	if !p.IsSuccess() || p.ExitCode() != 0 {
		return fmt.Errorf("failed to install kind: %s", p.Result())
	}

	// PATH may already be set to include $GOPATH/bin so we don't need to.
	if kindPath := k.executor.Prog().Avail("kind"); kindPath != "" {
		log.V(4).Info("Installed kind at", kindPath)
		return nil
	}

	p = k.executor.RunProc("ls $GOPATH/bin")
	if p.Err() != nil {
		return fmt.Errorf("failed to install kind: %s", p.Err())
	}

	p = k.executor.RunProc("echo $PATH:$GOPATH/bin")
	if p.Err() != nil {
		return fmt.Errorf("failed to install kind: %s", p.Err())
	}

	log.V(4).Info(`Setting path to include $GOPATH/bin:`, p.Result())
	k.executor.SetEnv("PATH", p.Result())

	if kindPath := k.executor.Prog().Avail("kind"); kindPath != "" {
		log.V(4).Info("Installed kind at", kindPath)
		return nil
	}
	return fmt.Errorf("kind not available even after installation")
}

func (k *kindCluster) Create(opts ...types.CreateOptions) (kubeConfig string, err error) {
	for _, opt := range opts {
		opt(k.cfg)
	}

	if err := k.installKindIfRequired(); err != nil {
		return "", err
	}
	if _, ok := k.clusterExists(k.cfg.Name); ok {
		log.V(4).Info("Skipping Kind Cluster.Create: cluster already created: ", k.cfg.Name)
		return k.KubeConfig()
	}

	command := fmt.Sprintf(`kind create cluster --name %s`, k.cfg.Name)
	if len(k.cfg.Args) > 0 {
		command = fmt.Sprintf("%s %s", command, strings.Join(k.cfg.Args, " "))
	}
	log.V(4).Info("Launching:", command)
	p := k.executor.RunProc(command)
	if p.Err() != nil {
		return "", fmt.Errorf("failed to create kind cluster: %s : %s", p.Err(), p.Result())
	}

	clusters, ok := k.clusterExists(k.cfg.Name)
	if !ok {
		return "", fmt.Errorf("kind Cluster.Create: cluster %v still not in 'cluster list' after creation: %v", k.cfg.Name, clusters)
	}
	log.V(4).Info("kind clusters available: ", clusters)

	return k.KubeConfig()
}

func (k *kindCluster) Destroy() (err error) {
	log.V(4).Info("Destroying kind cluster ", k.cfg.Name)
	if err := k.installKindIfRequired(); err != nil {
		return err
	}

	p := k.executor.RunProc(fmt.Sprintf(`kind delete cluster --name %s`, k.cfg.Name))
	if p.Err() != nil {
		return fmt.Errorf("kind: delete cluster failed: %s: %s", p.Err(), p.Result())
	}

	log.V(4).Info("Removing kubeconfig file ", k.kubeConfig)
	if err := os.RemoveAll(k.kubeConfig); err != nil {
		return fmt.Errorf("kind: remove kubefconfig failed: %w", err)
	}

	return nil
}

func (k *kindCluster) KubeConfig() (kubeConfig string, err error) {
	k.fetchOnce.Do(func() {
		err = k.saveKubeConfig()
	})
	if err != nil {
		return "", err
	}
	if k.kubeConfig == "" {
		return "", fmt.Errorf("failed to find kubeconfig file for cluster %s", k.cfg.Name)
	}
	return k.kubeConfig, nil
}

func (k *kindCluster) KubeCtx() (kubeCtx string) {
	return fmt.Sprintf("kind-%s", k.cfg.Name)
}

func (k *kindCluster) LoadImage(image string) (err error) {
	p := k.executor.RunProc(fmt.Sprintf(`kind load docker-image --name %s %s`, k.cfg.Name, image))
	if p.Err() != nil {
		return fmt.Errorf("kind: load docker-image failed: %s: %s", p.Err(), p.Result())
	}
	return nil
}

func (k *kindCluster) LoadImageArchive(archive string) (err error) {
	p := k.executor.RunProc(fmt.Sprintf(`kind load image-archive --name %s %s`, k.cfg.Name, archive))
	if p.Err() != nil {
		return fmt.Errorf("kind: load image-archive failed: %s: %s", p.Err(), p.Result())
	}
	return nil
}

func NewKindClusterProvider() types.ClusterProvider {
	return &kindCluster{
		cfg: &types.ClusterConfig{
			Args: make([]string, 0),
		},
		executor: gexe.New(),
	}
}

func init() {
	framework.RegisterProvider("kind", NewKindClusterProvider)
}
