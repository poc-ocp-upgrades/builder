package builder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	dockercmd "github.com/docker/docker/builder/dockerfile/command"
	"github.com/docker/docker/builder/dockerfile/parser"
	docker "github.com/fsouza/go-dockerclient"
	s2iapi "github.com/openshift/source-to-image/pkg/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	buildapiv1 "github.com/openshift/api/build/v1"
	"github.com/openshift/builder/pkg/build/builder/cmd/dockercfg"
	"github.com/openshift/builder/pkg/build/builder/timing"
	builderutil "github.com/openshift/builder/pkg/build/builder/util"
	"github.com/openshift/builder/pkg/build/builder/util/dockerfile"
	buildclientv1 "github.com/openshift/client-go/build/clientset/versioned/typed/build/v1"
)

const defaultDockerfilePath = "Dockerfile"

type DockerBuilder struct {
	dockerClient	DockerClient
	build		*buildapiv1.Build
	client		buildclientv1.BuildInterface
	cgLimits	*s2iapi.CGroupLimits
	inputDir	string
}

func NewDockerBuilder(dockerClient DockerClient, buildsClient buildclientv1.BuildInterface, build *buildapiv1.Build, cgLimits *s2iapi.CGroupLimits) *DockerBuilder {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return &DockerBuilder{dockerClient: dockerClient, build: build, client: buildsClient, cgLimits: cgLimits, inputDir: InputContentPath}
}
func (d *DockerBuilder) Build() error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	var err error
	ctx := timing.NewContext(context.Background())
	defer func() {
		d.build.Status.Stages = timing.AppendStageAndStepInfo(d.build.Status.Stages, timing.GetStages(ctx))
		HandleBuildStatusUpdate(d.build, d.client, nil)
	}()
	if d.build.Spec.Source.Git == nil && d.build.Spec.Source.Binary == nil && d.build.Spec.Source.Dockerfile == nil && d.build.Spec.Source.Images == nil {
		return fmt.Errorf("must provide a value for at least one of source, binary, images, or dockerfile")
	}
	var push bool
	pushTag := d.build.Status.OutputDockerImageReference
	buildDir := d.inputDir
	glog.V(4).Infof("Starting Docker build from build config %s ...", d.build.Name)
	if d.build.Spec.Output.To == nil || len(d.build.Spec.Output.To.Name) == 0 {
		d.build.Status.OutputDockerImageReference = d.build.Name
	} else {
		push = true
	}
	buildTag := randomBuildTag(d.build.Namespace, d.build.Name)
	dockerfilePath := getDockerfilePath(buildDir, d.build)
	imageNames, err := findReferencedImages(dockerfilePath)
	if err != nil {
		return err
	}
	if len(imageNames) == 0 {
		return fmt.Errorf("no FROM image in Dockerfile")
	}
	for _, imageName := range imageNames {
		if imageName == "scratch" {
			glog.V(4).Infof("\nSkipping image \"scratch\"")
			continue
		}
		imageExists := true
		_, err = d.dockerClient.InspectImage(imageName)
		if err != nil {
			if err != docker.ErrNoSuchImage {
				return err
			}
			imageExists = false
		}
		if d.build.Spec.Strategy.DockerStrategy.ForcePull || !imageExists {
			pullAuthConfig, _ := dockercfg.NewHelper().GetDockerAuth(imageName, dockercfg.PullAuthType)
			glog.V(0).Infof("\nPulling image %s ...", imageName)
			startTime := metav1.Now()
			err = d.pullImage(imageName, pullAuthConfig)
			timing.RecordNewStep(ctx, buildapiv1.StagePullImages, buildapiv1.StepPullBaseImage, startTime, metav1.Now())
			if err != nil {
				d.build.Status.Phase = buildapiv1.BuildPhaseFailed
				d.build.Status.Reason = buildapiv1.StatusReasonPullBuilderImageFailed
				d.build.Status.Message = builderutil.StatusMessagePullBuilderImageFailed
				HandleBuildStatusUpdate(d.build, d.client, nil)
				return fmt.Errorf("failed to pull image: %v", err)
			}
		}
	}
	startTime := metav1.Now()
	err = d.dockerBuild(ctx, buildDir, buildTag)
	timing.RecordNewStep(ctx, buildapiv1.StageBuild, buildapiv1.StepDockerBuild, startTime, metav1.Now())
	if err != nil {
		d.build.Status.Phase = buildapiv1.BuildPhaseFailed
		d.build.Status.Reason = buildapiv1.StatusReasonDockerBuildFailed
		d.build.Status.Message = builderutil.StatusMessageDockerBuildFailed
		HandleBuildStatusUpdate(d.build, d.client, nil)
		return err
	}
	if push {
		if err := tagImage(d.dockerClient, buildTag, pushTag); err != nil {
			return err
		}
	}
	if err := removeImage(d.dockerClient, buildTag); err != nil {
		glog.V(0).Infof("warning: Failed to remove temporary build tag %v: %v", buildTag, err)
	}
	if push && pushTag != "" {
		pushAuthConfig, authPresent := dockercfg.NewHelper().GetDockerAuth(pushTag, dockercfg.PushAuthType)
		if authPresent {
			glog.V(4).Infof("Authenticating Docker push with user %q", pushAuthConfig.Username)
		}
		glog.V(0).Infof("\nPushing image %s ...", pushTag)
		startTime = metav1.Now()
		digest, err := d.pushImage(pushTag, pushAuthConfig)
		timing.RecordNewStep(ctx, buildapiv1.StagePushImage, buildapiv1.StepPushDockerImage, startTime, metav1.Now())
		if err != nil {
			d.build.Status.Phase = buildapiv1.BuildPhaseFailed
			d.build.Status.Reason = buildapiv1.StatusReasonPushImageToRegistryFailed
			d.build.Status.Message = builderutil.StatusMessagePushImageToRegistryFailed
			HandleBuildStatusUpdate(d.build, d.client, nil)
			return reportPushFailure(err, authPresent, pushAuthConfig)
		}
		if len(digest) > 0 {
			d.build.Status.Output.To = &buildapiv1.BuildStatusOutputTo{ImageDigest: digest}
			HandleBuildStatusUpdate(d.build, d.client, nil)
		}
		glog.V(0).Infof("Push successful")
	}
	return nil
}
func (d *DockerBuilder) pullImage(name string, authConfig docker.AuthConfiguration) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	repository, tag := docker.ParseRepositoryTag(name)
	options := docker.PullImageOptions{Repository: repository, Tag: tag}
	if options.Tag == "" && strings.Contains(name, "@") {
		options.Repository = name
	}
	return retryImageAction("Pull", func() (pullErr error) {
		return d.dockerClient.PullImage(options, authConfig)
	})
}
func (d *DockerBuilder) pushImage(name string, authConfig docker.AuthConfiguration) (string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	repository, tag := docker.ParseRepositoryTag(name)
	options := docker.PushImageOptions{Name: repository, Tag: tag}
	var err error
	sha := ""
	retryImageAction("Push", func() (pushErr error) {
		sha, err = d.dockerClient.PushImage(options, authConfig)
		return err
	})
	return sha, err
}
func (d *DockerBuilder) copyConfigMaps(configs []buildapiv1.ConfigMapBuildSource, targetDir string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	var err error
	for _, c := range configs {
		err = d.copyLocalObject(configMapSource(c), configMapBuildSourceBaseMountPath, targetDir)
		if err != nil {
			return err
		}
	}
	return nil
}
func (d *DockerBuilder) copySecrets(secrets []buildapiv1.SecretBuildSource, targetDir string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	var err error
	for _, s := range secrets {
		err = d.copyLocalObject(secretSource(s), secretBuildSourceBaseMountPath, targetDir)
		if err != nil {
			return err
		}
	}
	return nil
}
func (d *DockerBuilder) copyLocalObject(s localObjectBuildSource, sourceDir, targetDir string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	dstDir := filepath.Join(targetDir, s.DestinationPath())
	if err := os.MkdirAll(dstDir, 0777); err != nil {
		return err
	}
	glog.V(3).Infof("Copying files from the build source %q to %q", s.LocalObjectRef().Name, dstDir)
	srcDir := filepath.Join(sourceDir, s.LocalObjectRef().Name)
	if err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if srcDir == path {
			return nil
		}
		if strings.HasPrefix(filepath.Base(path), "..") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() {
			if err := os.MkdirAll(dstDir, 0777); err != nil {
				return err
			}
		}
		out, err := exec.Command("cp", "-vLRf", path, dstDir+"/").Output()
		if err != nil {
			glog.V(4).Infof("Build source %q failed to copy: %q", s.LocalObjectRef().Name, string(out))
			return err
		}
		glog.V(5).Infof("Result of build source copy %s\n%s", s.LocalObjectRef().Name, string(out))
		return nil
	}); err != nil {
		return err
	}
	return nil
}
func (d *DockerBuilder) dockerBuild(ctx context.Context, dir string, tag string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	var noCache bool
	var forcePull bool
	var buildArgs []docker.BuildArg
	dockerfilePath := defaultDockerfilePath
	if d.build.Spec.Strategy.DockerStrategy != nil {
		if d.build.Spec.Source.ContextDir != "" {
			dir = filepath.Join(dir, d.build.Spec.Source.ContextDir)
		}
		if d.build.Spec.Strategy.DockerStrategy.DockerfilePath != "" {
			dockerfilePath = d.build.Spec.Strategy.DockerStrategy.DockerfilePath
		}
		for _, ba := range d.build.Spec.Strategy.DockerStrategy.BuildArgs {
			buildArgs = append(buildArgs, docker.BuildArg{Name: ba.Name, Value: ba.Value})
		}
		noCache = d.build.Spec.Strategy.DockerStrategy.NoCache
		forcePull = d.build.Spec.Strategy.DockerStrategy.ForcePull
	}
	var auth *docker.AuthConfigurations
	var err error
	path := os.Getenv(dockercfg.PullAuthType)
	if len(path) != 0 {
		auth, err = GetDockerAuthConfiguration(path)
		if err != nil {
			return err
		}
	}
	if err = d.copySecrets(d.build.Spec.Source.Secrets, dir); err != nil {
		return err
	}
	if err = d.copyConfigMaps(d.build.Spec.Source.ConfigMaps, dir); err != nil {
		return err
	}
	opts := docker.BuildImageOptions{Context: ctx, Name: tag, RmTmpContainer: true, ForceRmTmpContainer: true, OutputStream: os.Stdout, Dockerfile: dockerfilePath, NoCache: noCache, Pull: forcePull, BuildArgs: buildArgs, ContextDir: dir}
	if d.cgLimits != nil {
		opts.Memory = d.cgLimits.MemoryLimitBytes
		opts.Memswap = d.cgLimits.MemorySwap
		opts.CgroupParent = d.cgLimits.Parent
	}
	if auth != nil {
		opts.AuthConfigs = *auth
	}
	return d.dockerClient.BuildImage(opts)
}
func getDockerfilePath(dir string, build *buildapiv1.Build) string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	var contextDirPath string
	if build.Spec.Strategy.DockerStrategy != nil && len(build.Spec.Source.ContextDir) > 0 {
		contextDirPath = filepath.Join(dir, build.Spec.Source.ContextDir)
	} else {
		contextDirPath = dir
	}
	var dockerfilePath string
	if build.Spec.Strategy.DockerStrategy != nil && len(build.Spec.Strategy.DockerStrategy.DockerfilePath) > 0 {
		dockerfilePath = filepath.Join(contextDirPath, build.Spec.Strategy.DockerStrategy.DockerfilePath)
	} else {
		dockerfilePath = filepath.Join(contextDirPath, defaultDockerfilePath)
	}
	return dockerfilePath
}
func replaceLastFrom(node *parser.Node, image string, alias string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	if node == nil {
		return nil
	}
	for i := len(node.Children) - 1; i >= 0; i-- {
		child := node.Children[i]
		if child != nil && child.Value == dockercmd.From {
			if child.Next == nil {
				child.Next = &parser.Node{}
			}
			glog.Infof("Replaced Dockerfile FROM image %s", child.Next.Value)
			child.Next.Value = image
			if len(alias) != 0 {
				if child.Next.Next == nil {
					child.Next.Next = &parser.Node{}
				}
				child.Next.Next.Value = "as"
				if child.Next.Next.Next == nil {
					child.Next.Next.Next = &parser.Node{}
				}
				child.Next.Next.Next.Value = alias
			}
			return nil
		}
	}
	return nil
}
func getLastFrom(node *parser.Node) (string, string) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	if node == nil {
		return "", ""
	}
	var image, alias string
	for i := len(node.Children) - 1; i >= 0; i-- {
		child := node.Children[i]
		if child != nil && child.Value == dockercmd.From {
			if child.Next != nil {
				image = child.Next.Value
				if child.Next.Next != nil && strings.ToUpper(child.Next.Next.Value) == "AS" {
					if child.Next.Next.Next != nil {
						alias = child.Next.Next.Next.Value
					}
				}
				break
			}
		}
	}
	return image, alias
}
func appendEnv(node *parser.Node, m []dockerfile.KeyValue) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return appendKeyValueInstruction(dockerfile.Env, node, m)
}
func appendLabel(node *parser.Node, m []dockerfile.KeyValue) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	if len(m) == 0 {
		return nil
	}
	return appendKeyValueInstruction(dockerfile.Label, node, m)
}
func appendPostCommit(node *parser.Node, cmd string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	if len(cmd) == 0 {
		return nil
	}
	image, alias := getLastFrom(node)
	if len(alias) == 0 {
		alias = postCommitAlias
		if err := replaceLastFrom(node, image, alias); err != nil {
			return err
		}
	}
	if err := appendStringInstruction(dockerfile.From, node, alias); err != nil {
		return err
	}
	if err := appendStringInstruction(dockerfile.Run, node, cmd); err != nil {
		return err
	}
	if err := appendStringInstruction(dockerfile.From, node, alias); err != nil {
		return err
	}
	return nil
}
func appendStringInstruction(f func(string) (string, error), node *parser.Node, cmd string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	if node == nil {
		return nil
	}
	instruction, err := f(cmd)
	if err != nil {
		return err
	}
	return dockerfile.InsertInstructions(node, len(node.Children), instruction)
}
func appendKeyValueInstruction(f func([]dockerfile.KeyValue) (string, error), node *parser.Node, m []dockerfile.KeyValue) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	if node == nil {
		return nil
	}
	instruction, err := f(m)
	if err != nil {
		return err
	}
	return dockerfile.InsertInstructions(node, len(node.Children), instruction)
}
func insertEnvAfterFrom(node *parser.Node, env []corev1.EnvVar) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	if node == nil || len(env) == 0 {
		return nil
	}
	var m []dockerfile.KeyValue
	for _, e := range env {
		m = append(m, dockerfile.KeyValue{Key: e.Name, Value: e.Value})
	}
	buildEnv, err := dockerfile.Env(m)
	if err != nil {
		return err
	}
	indices := dockerfile.FindAll(node, dockercmd.From)
	for i := len(indices) - 1; i >= 0; i-- {
		err := dockerfile.InsertInstructions(node, indices[i]+1, buildEnv)
		if err != nil {
			return err
		}
	}
	return nil
}
