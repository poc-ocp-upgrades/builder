package builder

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
	"github.com/containers/buildah"
	"github.com/containers/buildah/imagebuildah"
	"github.com/containers/buildah/util"
	"github.com/containers/image/pkg/docker/config"
	"github.com/containers/image/transports/alltransports"
	"github.com/containers/image/types"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/archive"
	docker "github.com/fsouza/go-dockerclient"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	buildapiv1 "github.com/openshift/api/build/v1"
	"github.com/openshift/library-go/pkg/image/reference"
)

func pullDaemonlessImage(sc types.SystemContext, store storage.Store, imageName string, authConfig docker.AuthConfiguration, blobCacheDirectory string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	glog.V(2).Infof("Asked to pull fresh copy of %q.", imageName)
	if imageName == "" {
		return fmt.Errorf("unable to pull using empty image name")
	}
	_, err := alltransports.ParseImageName("docker://" + imageName)
	if err != nil {
		return fmt.Errorf("error parsing image name to pull %s: %v", "docker://"+imageName, err)
	}
	systemContext := sc
	systemContext.AuthFilePath = "/tmp/config.json"
	ref, err := reference.Parse(imageName)
	if err != nil {
		return fmt.Errorf("error parsing image name %s: %v", ref, err)
	}
	if ref.Registry == "" {
		glog.V(2).Infof("defaulting registry to docker.io for image %s", imageName)
		ref.Registry = "docker.io"
	}
	if authConfig.Username != "" && authConfig.Password != "" {
		glog.V(5).Infof("Setting authentication for registry %q for %q.", ref.Registry, imageName)
		if err := config.SetAuthentication(&systemContext, ref.Registry, authConfig.Username, authConfig.Password); err != nil {
			return err
		}
	}
	options := buildah.PullOptions{ReportWriter: os.Stderr, Store: store, SystemContext: &systemContext, BlobDirectory: blobCacheDirectory}
	return buildah.Pull(context.TODO(), "docker://"+imageName, options)
}
func buildDaemonlessImage(sc types.SystemContext, store storage.Store, isolation buildah.Isolation, contextDir string, optimization buildapiv1.ImageOptimizationPolicy, opts *docker.BuildImageOptions, blobCacheDirectory string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	glog.V(2).Infof("Building...")
	args := make(map[string]string)
	for _, ev := range opts.BuildArgs {
		args[ev.Name] = ev.Value
	}
	pullPolicy := buildah.PullIfMissing
	if opts.Pull {
		glog.V(2).Infof("Forcing fresh pull of base image.")
		pullPolicy = buildah.PullAlways
	}
	layers := false
	switch optimization {
	case buildapiv1.ImageOptimizationSkipLayers, buildapiv1.ImageOptimizationSkipLayersAndWarn:
	case buildapiv1.ImageOptimizationNone:
	default:
		return fmt.Errorf("internal error: image optimization policy %q not fully implemented", string(optimization))
	}
	systemContext := sc
	systemContext.AuthFilePath = "/tmp/config.json"
	for registry, ac := range opts.AuthConfigs.Configs {
		glog.V(5).Infof("Setting authentication for registry %q at %q.", registry, ac.ServerAddress)
		if err := config.SetAuthentication(&systemContext, registry, ac.Username, ac.Password); err != nil {
			return err
		}
		if err := config.SetAuthentication(&systemContext, ac.ServerAddress, ac.Username, ac.Password); err != nil {
			return err
		}
	}
	var transientMounts []imagebuildah.Mount
	if st, err := os.Stat("/run/secrets"); err == nil && st.IsDir() {
		transientMounts = append(transientMounts, imagebuildah.Mount{Source: "/run/secrets", Destination: "/run/secrets", Type: "bind", Options: []string{"ro", "nodev", "noexec", "nosuid"}})
	}
	options := imagebuildah.BuildOptions{ContextDirectory: contextDir, PullPolicy: pullPolicy, Isolation: isolation, TransientMounts: transientMounts, Args: args, Output: opts.Name, Out: opts.OutputStream, Err: opts.OutputStream, ReportWriter: opts.OutputStream, OutputFormat: buildah.Dockerv2ImageManifest, SystemContext: &systemContext, NamespaceOptions: buildah.NamespaceOptions{{Name: string(specs.NetworkNamespace), Host: true}}, CommonBuildOpts: &buildah.CommonBuildOptions{Memory: opts.Memory, MemorySwap: opts.Memswap, CgroupParent: opts.CgroupParent}, Layers: layers, NoCache: opts.NoCache, RemoveIntermediateCtrs: opts.RmTmpContainer, ForceRmIntermediateCtrs: true, BlobDirectory: blobCacheDirectory}
	_, _, err := imagebuildah.BuildDockerfiles(opts.Context, store, options, opts.Dockerfile)
	return err
}
func tagDaemonlessImage(sc types.SystemContext, store storage.Store, buildTag, pushTag string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	glog.V(2).Infof("Tagging local image %q with name %q.", buildTag, pushTag)
	if buildTag == "" {
		return fmt.Errorf("unable to add tag to image with empty image name")
	}
	if pushTag == "" {
		return fmt.Errorf("unable to add empty tag to image")
	}
	systemContext := sc
	_, img, err := util.FindImage(store, "", &systemContext, buildTag)
	if err != nil {
		return err
	}
	if img == nil {
		return storage.ErrImageUnknown
	}
	if err := store.SetNames(img.ID, append(img.Names, pushTag)); err != nil {
		return err
	}
	return nil
}
func removeDaemonlessImage(sc types.SystemContext, store storage.Store, buildTag string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	glog.V(2).Infof("Removing name %q from local image.", buildTag)
	if buildTag == "" {
		return fmt.Errorf("unable to remove image using empty image name")
	}
	systemContext := sc
	_, img, err := util.FindImage(store, "", &systemContext, buildTag)
	if err != nil {
		return err
	}
	if img == nil {
		return storage.ErrImageUnknown
	}
	filtered := make([]string, 0, len(img.Names))
	for _, name := range img.Names {
		if name != buildTag {
			filtered = append(filtered, name)
		}
	}
	if err := store.SetNames(img.ID, filtered); err != nil {
		return err
	}
	return nil
}
func pushDaemonlessImage(sc types.SystemContext, store storage.Store, imageName string, authConfig docker.AuthConfiguration, blobCacheDirectory string) (string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	glog.V(2).Infof("Pushing image %q from local storage.", imageName)
	if imageName == "" {
		return "", fmt.Errorf("unable to push using empty destination image name")
	}
	dest, err := alltransports.ParseImageName("docker://" + imageName)
	if err != nil {
		return "", fmt.Errorf("error parsing destination image name %s: %v", "docker://"+imageName, err)
	}
	systemContext := sc
	systemContext.AuthFilePath = "/tmp/config.json"
	if authConfig.Username != "" && authConfig.Password != "" {
		glog.V(2).Infof("Setting authentication secret for %q.", authConfig.ServerAddress)
		systemContext.DockerAuthConfig = &types.DockerAuthConfig{Username: authConfig.Username, Password: authConfig.Password}
	} else {
		glog.V(2).Infof("No authentication secret provided for pushing to registry.")
	}
	options := buildah.PushOptions{Compression: archive.Gzip, ReportWriter: os.Stdout, Store: store, SystemContext: &systemContext, BlobDirectory: blobCacheDirectory}
	_, digest, err := buildah.Push(context.TODO(), imageName, dest, options)
	return string(digest), err
}
func inspectDaemonlessImage(sc types.SystemContext, store storage.Store, name string) (*docker.Image, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	systemContext := sc
	ref, img, err := util.FindImage(store, "", &systemContext, name)
	if err != nil {
		switch errors.Cause(err) {
		case storage.ErrImageUnknown, docker.ErrNoSuchImage:
			glog.V(2).Infof("Local copy of %q is not present.", name)
			return nil, docker.ErrNoSuchImage
		}
		return nil, err
	}
	if img == nil {
		return nil, docker.ErrNoSuchImage
	}
	image, err := ref.NewImage(context.TODO(), &systemContext)
	if err != nil {
		return nil, err
	}
	defer image.Close()
	size, err := image.Size()
	if err != nil {
		return nil, err
	}
	oconfig, err := image.OCIConfig(context.TODO())
	if err != nil {
		return nil, err
	}
	var rootfs *docker.RootFS
	if len(oconfig.RootFS.DiffIDs) > 0 {
		rootfs = &docker.RootFS{Type: oconfig.RootFS.Type}
		for _, d := range oconfig.RootFS.DiffIDs {
			rootfs.Layers = append(rootfs.Layers, d.String())
		}
	}
	exposedPorts := make(map[docker.Port]struct{})
	for port := range oconfig.Config.ExposedPorts {
		exposedPorts[docker.Port(port)] = struct{}{}
	}
	config := docker.Config{User: oconfig.Config.User, ExposedPorts: exposedPorts, Env: oconfig.Config.Env, Entrypoint: oconfig.Config.Entrypoint, Cmd: oconfig.Config.Cmd, Volumes: oconfig.Config.Volumes, WorkingDir: oconfig.Config.WorkingDir, Labels: oconfig.Config.Labels, StopSignal: oconfig.Config.StopSignal}
	var created time.Time
	if oconfig.Created != nil {
		created = *oconfig.Created
	}
	return &docker.Image{ID: img.ID, RepoTags: []string{}, Parent: "", Comment: "", Created: created, Container: "", ContainerConfig: config, DockerVersion: "", Author: oconfig.Author, Config: &config, Architecture: oconfig.Architecture, Size: size, VirtualSize: size, RepoDigests: []string{}, RootFS: rootfs, OS: oconfig.OS}, nil
}
func daemonlessRun(ctx context.Context, store storage.Store, isolation buildah.Isolation, createOpts docker.CreateContainerOptions, attachOpts docker.AttachToContainerOptions, blobCacheDirectory string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	if createOpts.Config == nil {
		return fmt.Errorf("error calling daemonlessRun: expected a Config")
	}
	if createOpts.HostConfig == nil {
		return fmt.Errorf("error calling daemonlessRun: expected a HostConfig")
	}
	builderOptions := buildah.BuilderOptions{Container: createOpts.Name, FromImage: createOpts.Config.Image, CommonBuildOpts: &buildah.CommonBuildOptions{Memory: createOpts.HostConfig.Memory, MemorySwap: createOpts.HostConfig.MemorySwap, CgroupParent: createOpts.HostConfig.CgroupParent}, PullBlobDirectory: blobCacheDirectory}
	builder, err := buildah.NewBuilder(ctx, store, builderOptions)
	if err != nil {
		return err
	}
	defer func() {
		if err := builder.Delete(); err != nil {
			glog.V(0).Infof("Error deleting container %q(%s): %v", builder.Container, builder.ContainerID, err)
		}
	}()
	entrypoint := createOpts.Config.Entrypoint
	if len(entrypoint) == 0 {
		entrypoint = builder.Entrypoint()
	}
	runOptions := buildah.RunOptions{Isolation: isolation, Entrypoint: entrypoint, Cmd: createOpts.Config.Cmd, Stdout: attachOpts.OutputStream, Stderr: attachOpts.ErrorStream}
	return builder.Run(append(entrypoint, createOpts.Config.Cmd...), runOptions)
}

type DaemonlessClient struct {
	SystemContext		types.SystemContext
	Store			storage.Store
	Isolation		buildah.Isolation
	BlobCacheDirectory	string
	builders		map[string]*buildah.Builder
}

func GetDaemonlessClient(systemContext types.SystemContext, store storage.Store, isolationSpec, blobCacheDirectory string) (client DockerClient, err error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	isolation := buildah.IsolationDefault
	switch strings.ToLower(isolationSpec) {
	case "chroot":
		isolation = buildah.IsolationChroot
	case "oci":
		isolation = buildah.IsolationOCI
	case "rootless":
		isolation = buildah.IsolationOCIRootless
	case "":
	default:
		return nil, fmt.Errorf("unrecognized BUILD_ISOLATION setting %q", strings.ToLower(isolationSpec))
	}
	if blobCacheDirectory != "" {
		glog.V(0).Infof("Caching blobs under %q.", blobCacheDirectory)
	}
	return &DaemonlessClient{SystemContext: systemContext, Store: store, Isolation: isolation, BlobCacheDirectory: blobCacheDirectory, builders: make(map[string]*buildah.Builder)}, nil
}
func (d *DaemonlessClient) BuildImage(opts docker.BuildImageOptions) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return buildDaemonlessImage(d.SystemContext, d.Store, d.Isolation, opts.ContextDir, buildapiv1.ImageOptimizationNone, &opts, d.BlobCacheDirectory)
}
func (d *DaemonlessClient) PushImage(opts docker.PushImageOptions, auth docker.AuthConfiguration) (string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	imageName := opts.Name
	if opts.Tag != "" {
		imageName = imageName + ":" + opts.Tag
	}
	return pushDaemonlessImage(d.SystemContext, d.Store, imageName, auth, d.BlobCacheDirectory)
}
func (d *DaemonlessClient) RemoveImage(name string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return removeDaemonlessImage(d.SystemContext, d.Store, name)
}
func (d *DaemonlessClient) CreateContainer(opts docker.CreateContainerOptions) (*docker.Container, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	options := buildah.BuilderOptions{FromImage: opts.Config.Image, Container: opts.Name, PullBlobDirectory: d.BlobCacheDirectory}
	builder, err := buildah.NewBuilder(opts.Context, d.Store, options)
	if err != nil {
		return nil, err
	}
	builder.SetCmd(opts.Config.Cmd)
	builder.SetEntrypoint(opts.Config.Entrypoint)
	if builder.Container != "" {
		d.builders[builder.Container] = builder
	}
	if builder.ContainerID != "" {
		d.builders[builder.ContainerID] = builder
	}
	return &docker.Container{ID: builder.ContainerID}, nil
}
func (d *DaemonlessClient) RemoveContainer(opts docker.RemoveContainerOptions) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	builder, ok := d.builders[opts.ID]
	if !ok {
		return errors.Errorf("no such container as %q", opts.ID)
	}
	name := builder.Container
	id := builder.ContainerID
	err := builder.Delete()
	if err == nil {
		if name != "" {
			if _, ok := d.builders[name]; ok {
				delete(d.builders, name)
			}
		}
		if id != "" {
			if _, ok := d.builders[id]; ok {
				delete(d.builders, id)
			}
		}
	}
	return err
}
func (d *DaemonlessClient) PullImage(opts docker.PullImageOptions, auth docker.AuthConfiguration) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	imageName := opts.Repository
	if opts.Tag != "" {
		imageName = imageName + ":" + opts.Tag
	}
	return pullDaemonlessImage(d.SystemContext, d.Store, imageName, auth, d.BlobCacheDirectory)
}
func (d *DaemonlessClient) TagImage(name string, opts docker.TagImageOptions) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	imageName := opts.Repo
	if opts.Tag != "" {
		imageName = imageName + ":" + opts.Tag
	}
	return tagDaemonlessImage(d.SystemContext, d.Store, name, imageName)
}
func (d *DaemonlessClient) InspectImage(name string) (*docker.Image, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return inspectDaemonlessImage(d.SystemContext, d.Store, name)
}
