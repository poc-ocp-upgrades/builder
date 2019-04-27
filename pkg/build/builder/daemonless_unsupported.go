package builder

import (
	"context"
	"github.com/containers/image/types"
	"github.com/containers/storage"
	docker "github.com/fsouza/go-dockerclient"
	buildapiv1 "github.com/openshift/api/build/v1"
	"github.com/pkg/errors"
)

type Isolation struct{}
type DummyStore struct{}
type DaemonlessClient struct {
	Isolation	Isolation
	Store		storage.Store
	SystemContext	types.SystemContext
}

func (d *DaemonlessClient) BuildImage(opts docker.BuildImageOptions) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return errors.New("building images not supported on this platform")
}
func (d *DaemonlessClient) PushImage(opts docker.PushImageOptions, auth docker.AuthConfiguration) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return errors.New("pushing images not supported on this platform")
}
func (d *DaemonlessClient) RemoveImage(name string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return errors.New("removing images not supported on this platform")
}
func (d *DaemonlessClient) CreateContainer(opts docker.CreateContainerOptions) (*docker.Container, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return nil, errors.New("creating containers not supported on this platform")
}
func (d *DaemonlessClient) PullImage(opts docker.PullImageOptions, auth docker.AuthConfiguration) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return errors.New("pulling images not supported on this platform")
}
func (d *DaemonlessClient) RemoveContainer(opts docker.RemoveContainerOptions) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return errors.New("removing containers not supported on this platform")
}
func (d *DaemonlessClient) InspectImage(name string) (*docker.Image, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return nil, errors.New("inspecting images not supported on this platform")
}
func (d *DaemonlessClient) TagImage(name string, opts docker.TagImageOptions) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return errors.New("tagging images not supported on this platform")
}
func daemonlessRun(ctx context.Context, store storage.Store, isolation Isolation, createOpts docker.CreateContainerOptions, attachOpts docker.AttachToContainerOptions) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return errors.New("running containers not supported on this platform")
}
func buildDaemonlessImage(sc types.SystemContext, store storage.Store, isolation Isolation, dir string, optimization buildapiv1.ImageOptimizationPolicy, opts *docker.BuildImageOptions) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return errors.New("running building images not supported on this platform")
}
func GetDaemonlessClient(systemContext types.SystemContext, store storage.Store, isolationSpec, blobCacheDirectory string) (client DockerClient, err error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return nil, errors.New("building images without an engine not supported on this platform")
}
