package builder

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	dockerclient "github.com/fsouza/go-dockerclient"
	s2iapi "github.com/openshift/source-to-image/pkg/api"
	s2iconstants "github.com/openshift/source-to-image/pkg/api/constants"
	"github.com/openshift/source-to-image/pkg/api/describe"
	"github.com/openshift/source-to-image/pkg/api/validation"
	s2ibuild "github.com/openshift/source-to-image/pkg/build"
	s2i "github.com/openshift/source-to-image/pkg/build/strategies"
	"github.com/openshift/source-to-image/pkg/docker"
	s2igit "github.com/openshift/source-to-image/pkg/scm/git"
	s2iutil "github.com/openshift/source-to-image/pkg/util"
	buildapiv1 "github.com/openshift/api/build/v1"
	"github.com/openshift/builder/pkg/build/builder/cmd/dockercfg"
	"github.com/openshift/builder/pkg/build/builder/timing"
	builderutil "github.com/openshift/builder/pkg/build/builder/util"
	"github.com/openshift/builder/pkg/build/builder/util/dockerfile"
	buildclientv1 "github.com/openshift/client-go/build/clientset/versioned/typed/build/v1"
	"github.com/openshift/imagebuilder"
	"github.com/openshift/library-go/pkg/git"
	"github.com/openshift/origin/pkg/build/apis/build"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type builderFactory interface {
	Builder(config *s2iapi.Config, overrides s2ibuild.Overrides) (s2ibuild.Builder, s2iapi.BuildInfo, error)
}
type validator interface {
	ValidateConfig(config *s2iapi.Config) []validation.Error
}
type runtimeBuilderFactory struct{ dockerClient DockerClient }

func (r runtimeBuilderFactory) Builder(config *s2iapi.Config, overrides s2ibuild.Overrides) (s2ibuild.Builder, s2iapi.BuildInfo, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	var client docker.Client
	var err error
	builder, buildInfo, err := s2i.Strategy(client, config, overrides)
	return builder, buildInfo, err
}

type runtimeConfigValidator struct{}

func (_ runtimeConfigValidator) ValidateConfig(config *s2iapi.Config) []validation.Error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return validation.ValidateConfig(config)
}

type S2IBuilder struct {
	builder		builderFactory
	validator	validator
	dockerClient	DockerClient
	dockerSocket	string
	build		*buildapiv1.Build
	client		buildclientv1.BuildInterface
	cgLimits	*s2iapi.CGroupLimits
}

func NewS2IBuilder(dockerClient DockerClient, dockerSocket string, buildsClient buildclientv1.BuildInterface, build *buildapiv1.Build, cgLimits *s2iapi.CGroupLimits) *S2IBuilder {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return newS2IBuilder(dockerClient, dockerSocket, buildsClient, build, runtimeBuilderFactory{dockerClient}, runtimeConfigValidator{}, cgLimits)
}
func newS2IBuilder(dockerClient DockerClient, dockerSocket string, buildsClient buildclientv1.BuildInterface, build *buildapiv1.Build, builder builderFactory, validator validator, cgLimits *s2iapi.CGroupLimits) *S2IBuilder {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return &S2IBuilder{builder: builder, validator: validator, dockerClient: dockerClient, dockerSocket: dockerSocket, build: build, client: buildsClient, cgLimits: cgLimits}
}
func injectConfigMaps(configMaps []buildapiv1.ConfigMapBuildSource) []s2iapi.VolumeSpec {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	vols := make([]s2iapi.VolumeSpec, len(configMaps))
	for i, c := range configMaps {
		vols[i] = makeVolumeSpec(configMapSource(c), configMapBuildSourceBaseMountPath)
	}
	return vols
}
func injectSecrets(secrets []buildapiv1.SecretBuildSource) []s2iapi.VolumeSpec {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	vols := make([]s2iapi.VolumeSpec, len(secrets))
	for i, s := range secrets {
		vols[i] = makeVolumeSpec(secretSource(s), secretBuildSourceBaseMountPath)
	}
	return vols
}
func makeVolumeSpec(src localObjectBuildSource, mountPath string) s2iapi.VolumeSpec {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	glog.V(3).Infof("Injecting build source %q into a build into %q", src.LocalObjectRef().Name, filepath.Clean(src.DestinationPath()))
	return s2iapi.VolumeSpec{Source: filepath.Join(mountPath, src.LocalObjectRef().Name), Destination: src.DestinationPath(), Keep: !src.IsSecret()}
}
func (s *S2IBuilder) Build() error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	var err error
	ctx := timing.NewContext(context.Background())
	defer func() {
		s.build.Status.Stages = timing.AppendStageAndStepInfo(s.build.Status.Stages, timing.GetStages(ctx))
		HandleBuildStatusUpdate(s.build, s.client, nil)
	}()
	if s.build.Spec.Strategy.SourceStrategy == nil {
		return errors.New("the source to image builder must be used with the source strategy")
	}
	var push bool
	if s.build.Spec.Output.To == nil || len(s.build.Spec.Output.To.Name) == 0 {
		s.build.Status.OutputDockerImageReference = s.build.Name
	} else {
		push = true
	}
	pushTag := s.build.Status.OutputDockerImageReference
	sourceInfo, err := readSourceInfo()
	if err != nil {
		return fmt.Errorf("error reading git source info: %v", err)
	}
	var s2iSourceInfo *s2igit.SourceInfo
	if sourceInfo != nil {
		s2iSourceInfo = toS2ISourceInfo(sourceInfo)
	}
	injections := s2iapi.VolumeList{}
	injections = append(injections, injectSecrets(s.build.Spec.Source.Secrets)...)
	injections = append(injections, injectConfigMaps(s.build.Spec.Source.ConfigMaps)...)
	buildTag := randomBuildTag(s.build.Namespace, s.build.Name)
	scriptDownloadProxyConfig, err := scriptProxyConfig(s.build)
	if err != nil {
		return err
	}
	if scriptDownloadProxyConfig != nil {
		glog.V(0).Infof("Using HTTP proxy %v and HTTPS proxy %v for script download", builderutil.SafeForLoggingURL(scriptDownloadProxyConfig.HTTPProxy), builderutil.SafeForLoggingURL(scriptDownloadProxyConfig.HTTPSProxy))
	}
	var incremental bool
	if s.build.Spec.Strategy.SourceStrategy.Incremental != nil {
		incremental = *s.build.Spec.Strategy.SourceStrategy.Incremental
	}
	srcDir := InputContentPath
	contextDir := ""
	if len(s.build.Spec.Source.ContextDir) != 0 {
		contextDir = filepath.Clean(s.build.Spec.Source.ContextDir)
		if contextDir == "." || contextDir == "/" {
			contextDir = ""
		}
	}
	config := &s2iapi.Config{PreserveWorkingDir: true, WorkingDir: "/tmp", DockerConfig: &s2iapi.DockerConfig{Endpoint: s.dockerSocket}, DockerCfgPath: os.Getenv(dockercfg.PullAuthType), LabelNamespace: builderutil.DefaultDockerLabelNamespace, ScriptsURL: s.build.Spec.Strategy.SourceStrategy.Scripts, BuilderImage: s.build.Spec.Strategy.SourceStrategy.From.Name, BuilderPullPolicy: s2iapi.PullAlways, Incremental: incremental, IncrementalFromTag: pushTag, Environment: buildEnvVars(s.build, sourceInfo), Labels: s2iBuildLabels(s.build, sourceInfo), Source: &s2igit.URL{URL: url.URL{Path: srcDir}, Type: s2igit.URLTypeLocal}, ContextDir: contextDir, SourceInfo: s2iSourceInfo, ForceCopy: true, Injections: injections, AsDockerfile: "/tmp/dockercontext/Dockerfile", ScriptDownloadProxyConfig: scriptDownloadProxyConfig, BlockOnBuild: true, KeepSymlinks: true}
	t, _ := dockercfg.NewHelper().GetDockerAuth(config.BuilderImage, dockercfg.PullAuthType)
	config.PullAuthentication = s2iapi.AuthConfig{Username: t.Username, Password: t.Password, Email: t.Email, ServerAddress: t.ServerAddress}
	if s.build.Spec.Strategy.SourceStrategy.ForcePull || !isImagePresent(s.dockerClient, config.BuilderImage) {
		startTime := metav1.Now()
		err = s.pullImage(config.BuilderImage, t)
		timing.RecordNewStep(ctx, buildapiv1.StagePullImages, buildapiv1.StepPullBaseImage, startTime, metav1.Now())
		if err != nil {
			return err
		}
	}
	if config.Incremental {
		if s.build.Spec.Strategy.SourceStrategy.ForcePull || !isImagePresent(s.dockerClient, config.IncrementalFromTag) {
			startTime := metav1.Now()
			err = s.pullImage(config.IncrementalFromTag, t)
			timing.RecordNewStep(ctx, buildapiv1.StagePullImages, buildapiv1.StepPullInputImage, startTime, metav1.Now())
			if err != nil {
				glog.V(2).Infof("Failed to pull incremental builder image %s - executing normal s2i build instead.", config.IncrementalFromTag)
				glog.V(5).Infof("Incremental image pull failure: %v", err)
				config.Incremental = false
				config.IncrementalFromTag = ""
			}
		}
	}
	assembleUser, err := getAssembleUser(s.dockerClient, config.BuilderImage)
	if err != nil {
		return err
	}
	if len(assembleUser) > 0 {
		glog.V(4).Infof("Using builder image assemble user %s", assembleUser)
		config.AssembleUser = assembleUser
	}
	labels, err := getImageLabels(s.dockerClient, config.BuilderImage)
	if err != nil {
		return err
	}
	destination := labels[s2iconstants.DestinationLabel]
	if len(destination) > 0 {
		glog.V(4).Infof("Using builder image destination %s", destination)
		config.Destination = destination
	}
	if len(config.ScriptsURL) == 0 {
		scriptsURL := labels[s2iconstants.ScriptsURLLabel]
		if len(scriptsURL) > 0 {
			glog.V(4).Infof("Using builder scripts URL %s", destination)
			config.ImageScriptsURL = scriptsURL
		}
	}
	allowedUIDs := os.Getenv(builderutil.AllowedUIDs)
	glog.V(4).Infof("The value of %s is [%s]", builderutil.AllowedUIDs, allowedUIDs)
	if len(allowedUIDs) > 0 {
		err = config.AllowedUIDs.Set(allowedUIDs)
		if err != nil {
			return err
		}
	}
	if errs := s.validator.ValidateConfig(config); len(errs) != 0 {
		var buffer bytes.Buffer
		for _, ve := range errs {
			buffer.WriteString(ve.Error())
			buffer.WriteString(", ")
		}
		return errors.New(buffer.String())
	}
	if glog.Is(4) {
		redactedConfig := SafeForLoggingS2IConfig(config)
		glog.V(4).Infof("Creating a new S2I builder with config: %#v\n", describe.Config(nil, redactedConfig))
	}
	builder, buildInfo, err := s.builder.Builder(config, s2ibuild.Overrides{Downloader: nil})
	if err != nil {
		s.build.Status.Phase = buildapiv1.BuildPhaseFailed
		s.build.Status.Reason, s.build.Status.Message = convertS2IFailureType(buildInfo.FailureReason.Reason, buildInfo.FailureReason.Message)
		HandleBuildStatusUpdate(s.build, s.client, nil)
		return err
	}
	glog.V(4).Infof("Starting S2I build from %s/%s BuildConfig ...", s.build.Namespace, s.build.Name)
	glog.Infof("Generating dockerfile with builder image %s", s.build.Spec.Strategy.SourceStrategy.From.Name)
	result, err := builder.Build(config)
	for _, stage := range result.BuildInfo.Stages {
		for _, step := range stage.Steps {
			timing.RecordNewStep(ctx, buildapiv1.StageName(stage.Name), buildapiv1.StepName(step.Name), metav1.NewTime(step.StartTime), metav1.NewTime(step.StartTime.Add(time.Duration(step.DurationMilliseconds)*time.Millisecond)))
		}
	}
	if err != nil {
		s.build.Status.Phase = buildapiv1.BuildPhaseFailed
		if result != nil {
			s.build.Status.Reason, s.build.Status.Message = convertS2IFailureType(result.BuildInfo.FailureReason.Reason, result.BuildInfo.FailureReason.Message)
		} else {
			s.build.Status.Reason = buildapiv1.StatusReasonGenericBuildFailed
			s.build.Status.Message = build.StatusMessageGenericBuildFailed
		}
		HandleBuildStatusUpdate(s.build, s.client, nil)
		return err
	}
	opts := dockerclient.BuildImageOptions{Context: ctx, Name: buildTag, RmTmpContainer: true, ForceRmTmpContainer: true, OutputStream: os.Stdout, Dockerfile: defaultDockerfilePath, NoCache: false, Pull: s.build.Spec.Strategy.SourceStrategy.ForcePull, ContextDir: "/tmp/dockercontext"}
	if s.cgLimits != nil {
		opts.CPUPeriod = s.cgLimits.CPUPeriod
		opts.CPUQuota = s.cgLimits.CPUQuota
		opts.CPUShares = s.cgLimits.CPUShares
		opts.Memory = s.cgLimits.MemoryLimitBytes
		opts.Memswap = s.cgLimits.MemorySwap
		opts.CgroupParent = s.cgLimits.Parent
	}
	pullAuthConfigs, err := s.setupPullSecret()
	if err != nil {
		s.build.Status.Phase = buildapiv1.BuildPhaseFailed
		s.build.Status.Reason = buildapiv1.StatusReasonPullBuilderImageFailed
		s.build.Status.Message = builderutil.StatusMessagePullBuilderImageFailed
		return err
	}
	if pullAuthConfigs != nil {
		opts.AuthConfigs = *pullAuthConfigs
	}
	startTime := metav1.Now()
	if _, err := os.Stat(config.AsDockerfile); !os.IsNotExist(err) {
		in, err := ioutil.ReadFile(config.AsDockerfile)
		if err != nil {
			return err
		}
		node, err := imagebuilder.ParseDockerfile(bytes.NewBuffer(in))
		if err != nil {
			return err
		}
		if err := appendPostCommit(node, buildPostCommit(s.build.Spec.PostCommit)); err != nil {
			return err
		}
		out := dockerfile.Write(node)
		glog.V(4).Infof("Replacing dockerfile\n%s\nwith:\n%s", string(in), string(out))
		overwriteFile(config.AsDockerfile, out)
	}
	err = s.dockerClient.BuildImage(opts)
	timing.RecordNewStep(ctx, buildapiv1.StageBuild, buildapiv1.StepDockerBuild, startTime, metav1.Now())
	if err != nil {
		s.build.Status.Phase = buildapiv1.BuildPhaseFailed
		s.build.Status.Reason = buildapiv1.StatusReasonGenericBuildFailed
		s.build.Status.Message = builderutil.StatusMessageGenericBuildFailed
		return err
	}
	if push {
		if err = tagImage(s.dockerClient, buildTag, pushTag); err != nil {
			return err
		}
		pushAuthConfig, authPresent := dockercfg.NewHelper().GetDockerAuth(pushTag, dockercfg.PushAuthType)
		if authPresent {
			glog.V(3).Infof("Using provided push secret for pushing %s image", pushTag)
		} else {
			glog.V(3).Infof("No push secret provided")
		}
		glog.V(0).Infof("\nPushing image %s ...", pushTag)
		startTime := metav1.Now()
		digest, err := s.pushImage(pushTag, pushAuthConfig)
		timing.RecordNewStep(ctx, buildapiv1.StagePushImage, buildapiv1.StepPushImage, startTime, metav1.Now())
		if err != nil {
			s.build.Status.Phase = buildapiv1.BuildPhaseFailed
			s.build.Status.Reason = buildapiv1.StatusReasonPushImageToRegistryFailed
			s.build.Status.Message = builderutil.StatusMessagePushImageToRegistryFailed
			HandleBuildStatusUpdate(s.build, s.client, nil)
			return reportPushFailure(err, authPresent, pushAuthConfig)
		}
		if len(digest) > 0 {
			s.build.Status.Output.To = &buildapiv1.BuildStatusOutputTo{ImageDigest: digest}
			HandleBuildStatusUpdate(s.build, s.client, nil)
		}
		glog.V(0).Infof("Push successful")
	}
	return nil
}
func (s *S2IBuilder) setupPullSecret() (*dockerclient.AuthConfigurations, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	if len(os.Getenv(dockercfg.PullAuthType)) == 0 {
		return nil, nil
	}
	glog.V(2).Infof("Checking for Docker config file for %s in path %s", dockercfg.PullAuthType, os.Getenv(dockercfg.PullAuthType))
	dockercfgPath := dockercfg.GetDockercfgFile(os.Getenv(dockercfg.PullAuthType))
	if len(dockercfgPath) == 0 {
		return nil, fmt.Errorf("no docker config file found in '%s'", os.Getenv(dockercfg.PullAuthType))
	}
	glog.V(2).Infof("Using Docker config file %s", dockercfgPath)
	r, err := os.Open(dockercfgPath)
	if err != nil {
		return nil, fmt.Errorf("'%s': %s", dockercfgPath, err)
	}
	return dockerclient.NewAuthConfigurations(r)
}
func (s *S2IBuilder) pullImage(name string, authConfig dockerclient.AuthConfiguration) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	glog.V(2).Infof("Explicitly pulling image %s", name)
	repository, tag := dockerclient.ParseRepositoryTag(name)
	options := dockerclient.PullImageOptions{Repository: repository, Tag: tag}
	if options.Tag == "" && strings.Contains(name, "@") {
		options.Repository = name
	}
	return retryImageAction("Pull", func() (pullErr error) {
		return s.dockerClient.PullImage(options, authConfig)
	})
}
func (s *S2IBuilder) buildImage(optimization buildapiv1.ImageOptimizationPolicy, opts dockerclient.BuildImageOptions) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return s.dockerClient.BuildImage(opts)
}
func (s *S2IBuilder) pushImage(name string, authConfig dockerclient.AuthConfiguration) (string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	repository, tag := dockerclient.ParseRepositoryTag(name)
	options := dockerclient.PushImageOptions{Name: repository, Tag: tag}
	var err error
	sha := ""
	retryImageAction("Push", func() (pushErr error) {
		sha, err = s.dockerClient.PushImage(options, authConfig)
		return err
	})
	return sha, err
}
func buildEnvVars(build *buildapiv1.Build, sourceInfo *git.SourceInfo) s2iapi.EnvironmentList {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	bi := buildInfo(build, sourceInfo)
	envVars := &s2iapi.EnvironmentList{}
	for _, item := range bi {
		envVars.Set(fmt.Sprintf("%s=%s", item.Key, item.Value))
	}
	return *envVars
}
func s2iBuildLabels(build *buildapiv1.Build, sourceInfo *git.SourceInfo) map[string]string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	labels := map[string]string{}
	if sourceInfo == nil {
		sourceInfo = &git.SourceInfo{}
	}
	if len(build.Spec.Source.ContextDir) > 0 {
		sourceInfo.ContextDir = build.Spec.Source.ContextDir
	}
	labels = s2iutil.GenerateLabelsFromSourceInfo(labels, toS2ISourceInfo(sourceInfo), builderutil.DefaultDockerLabelNamespace)
	if build != nil && build.Spec.Source.Git != nil && build.Spec.Source.Git.Ref != "" {
		labels[builderutil.DefaultDockerLabelNamespace+"build.commit.ref"] = build.Spec.Source.Git.Ref
	}
	for _, lbl := range build.Spec.Output.ImageLabels {
		labels[lbl.Name] = lbl.Value
	}
	return labels
}
func scriptProxyConfig(build *buildapiv1.Build) (*s2iapi.ProxyConfig, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	httpProxy := ""
	httpsProxy := ""
	for _, env := range build.Spec.Strategy.SourceStrategy.Env {
		switch env.Name {
		case "HTTP_PROXY", "http_proxy":
			httpProxy = env.Value
		case "HTTPS_PROXY", "https_proxy":
			httpsProxy = env.Value
		}
	}
	if len(httpProxy) == 0 && len(httpsProxy) == 0 {
		return nil, nil
	}
	config := &s2iapi.ProxyConfig{}
	if len(httpProxy) > 0 {
		proxyURL, err := ParseProxyURL(httpProxy)
		if err != nil {
			return nil, err
		}
		config.HTTPProxy = proxyURL
	}
	if len(httpsProxy) > 0 {
		proxyURL, err := ParseProxyURL(httpsProxy)
		if err != nil {
			return nil, err
		}
		config.HTTPSProxy = proxyURL
	}
	return config, nil
}
func copyToVolumeList(artifactsMapping []buildapiv1.ImageSourcePath) (volumeList s2iapi.VolumeList) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	for _, mappedPath := range artifactsMapping {
		volumeList = append(volumeList, s2iapi.VolumeSpec{Source: mappedPath.SourcePath, Destination: mappedPath.DestinationDir})
	}
	return
}
func convertS2IFailureType(reason s2iapi.StepFailureReason, message s2iapi.StepFailureMessage) (buildapiv1.StatusReason, string) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return buildapiv1.StatusReason(reason), string(message)
}
func isImagePresent(docker DockerClient, imageTag string) bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	image, err := docker.InspectImage(imageTag)
	return err == nil && image != nil
}
func getImageLabels(docker DockerClient, imageTag string) (map[string]string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	image, err := docker.InspectImage(imageTag)
	if err != nil {
		return nil, err
	}
	return image.ContainerConfig.Labels, nil
}
func getAssembleUser(docker DockerClient, imageTag string) (string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	image, err := docker.InspectImage(imageTag)
	if err != nil {
		return "", err
	}
	assembleUser := image.ContainerConfig.User
	if labelAssembleUser, ok := image.ContainerConfig.Labels[s2iconstants.AssembleUserLabel]; ok {
		assembleUser = labelAssembleUser
	}
	return assembleUser, nil
}
