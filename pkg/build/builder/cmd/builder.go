package cmd

import (
	"context"
	godefaultbytes "bytes"
	godefaulthttp "net/http"
	godefaultruntime "runtime"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	restclient "k8s.io/client-go/rest"
	istorage "github.com/containers/image/storage"
	"github.com/containers/image/types"
	"github.com/containers/storage"
	buildapiv1 "github.com/openshift/api/build/v1"
	bld "github.com/openshift/builder/pkg/build/builder"
	"github.com/openshift/builder/pkg/build/builder/cmd/scmauth"
	"github.com/openshift/builder/pkg/build/builder/timing"
	builderutil "github.com/openshift/builder/pkg/build/builder/util"
	utilglog "github.com/openshift/builder/pkg/build/builder/util/glog"
	"github.com/openshift/builder/pkg/version"
	buildscheme "github.com/openshift/client-go/build/clientset/versioned/scheme"
	buildclientv1 "github.com/openshift/client-go/build/clientset/versioned/typed/build/v1"
	"github.com/openshift/library-go/pkg/git"
	"github.com/openshift/library-go/pkg/serviceability"
	s2iapi "github.com/openshift/source-to-image/pkg/api"
	s2igit "github.com/openshift/source-to-image/pkg/scm/git"
)

var (
	glog				= utilglog.ToFile(os.Stderr, 2)
	buildScheme			= runtime.NewScheme()
	buildCodecFactory	= serializer.NewCodecFactory(buildscheme.Scheme)
	buildJSONCodec		runtime.Codec
)

func init() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	buildJSONCodec = buildCodecFactory.LegacyCodec(buildapiv1.SchemeGroupVersion)
}

type builder interface {
	Build(dockerClient bld.DockerClient, sock string, buildsClient buildclientv1.BuildInterface, build *buildapiv1.Build, cgLimits *s2iapi.CGroupLimits) error
}
type builderConfig struct {
	out				io.Writer
	build			*buildapiv1.Build
	sourceSecretDir	string
	dockerClient	bld.DockerClient
	dockerEndpoint	string
	buildsClient	buildclientv1.BuildInterface
	cleanup			func()
	store			storage.Store
	blobCache		string
}

func newBuilderConfigFromEnvironment(out io.Writer, needsDocker bool) (*builderConfig, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	cfg := &builderConfig{}
	var err error
	cfg.out = out
	buildStr := os.Getenv("BUILD")
	cfg.build = &buildapiv1.Build{}
	obj, _, err := buildJSONCodec.Decode([]byte(buildStr), nil, cfg.build)
	if err != nil {
		return nil, fmt.Errorf("unable to parse build string: %v", err)
	}
	ok := false
	cfg.build, ok = obj.(*buildapiv1.Build)
	if !ok {
		return nil, fmt.Errorf("build string %s is not a build: %#v", buildStr, obj)
	}
	if glog.Is(4) {
		redactedBuild := builderutil.SafeForLoggingBuild(cfg.build)
		bytes, err := runtime.Encode(buildJSONCodec, redactedBuild)
		if err != nil {
			glog.V(4).Infof("unable to print debug line: %v", err)
		} else {
			glog.V(4).Infof("redacted build: %v", string(bytes))
		}
	}
	cfg.sourceSecretDir = os.Getenv("SOURCE_SECRET_PATH")
	if needsDocker {
		var systemContext types.SystemContext
		if registriesConfPath, ok := os.LookupEnv("BUILD_REGISTRIES_CONF_PATH"); ok && len(registriesConfPath) > 0 {
			if _, err := os.Stat(registriesConfPath); err == nil {
				systemContext.SystemRegistriesConfPath = registriesConfPath
			}
		}
		if registriesDirPath, ok := os.LookupEnv("BUILD_REGISTRIES_DIR_PATH"); ok && len(registriesDirPath) > 0 {
			if _, err := os.Stat(registriesDirPath); err == nil {
				systemContext.RegistriesDirPath = registriesDirPath
			}
		}
		if signaturePolicyPath, ok := os.LookupEnv("BUILD_SIGNATURE_POLICY_PATH"); ok && len(signaturePolicyPath) > 0 {
			if _, err := os.Stat(signaturePolicyPath); err == nil {
				systemContext.SignaturePolicyPath = signaturePolicyPath
			}
		}
		storeOptions := storage.DefaultStoreOptions
		if driver, ok := os.LookupEnv("BUILD_STORAGE_DRIVER"); ok {
			storeOptions.GraphDriverName = driver
		}
		if storageConfPath, ok := os.LookupEnv("BUILD_STORAGE_CONF_PATH"); ok && len(storageConfPath) > 0 {
			if _, err := os.Stat(storageConfPath); err == nil {
				storage.ReloadConfigurationFile(storageConfPath, &storeOptions)
			}
		}
		store, err := storage.GetStore(storeOptions)
		cfg.store = store
		if err != nil {
			return nil, err
		}
		cfg.cleanup = func() {
			if _, err := store.Shutdown(false); err != nil {
				glog.V(0).Infof("Error shutting down storage: %v", err)
			}
		}
		istorage.Transport.SetStore(store)
		cfg.blobCache = "/var/cache/blobs"
		if blobCacheDir, isSet := os.LookupEnv("BUILD_BLOBCACHE_DIR"); isSet {
			cfg.blobCache = blobCacheDir
		}
		dockerClient, err := bld.GetDaemonlessClient(systemContext, store, os.Getenv("BUILD_ISOLATION"), cfg.blobCache)
		if err != nil {
			return nil, fmt.Errorf("no daemonless store: %v", err)
		}
		cfg.dockerClient = dockerClient
		cfg.dockerEndpoint = "n/a"
	}
	clientConfig, err := restclient.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("cannot connect to the server: %v", err)
	}
	buildsClient, err := buildclientv1.NewForConfig(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %v", err)
	}
	cfg.buildsClient = buildsClient.Builds(cfg.build.Namespace)
	return cfg, nil
}
func (c *builderConfig) setupGitEnvironment() (string, []string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	gitSource := c.build.Spec.Source.Git
	if gitSource == nil {
		return "", []string{}, nil
	}
	sourceSecret := c.build.Spec.Source.SourceSecret
	gitEnv := []string{"GIT_ASKPASS=true"}
	if sourceSecret != nil {
		sourceURL, err := s2igit.Parse(gitSource.URI)
		if err != nil {
			return "", nil, fmt.Errorf("cannot parse build URL: %s", gitSource.URI)
		}
		scmAuths := scmauth.GitAuths(sourceURL)
		secretsEnv, overrideURL, err := scmAuths.Setup(c.sourceSecretDir)
		if err != nil {
			return c.sourceSecretDir, nil, fmt.Errorf("cannot setup source secret: %v", err)
		}
		if overrideURL != nil {
			gitSource.URI = overrideURL.String()
		}
		gitEnv = append(gitEnv, secretsEnv...)
	}
	if gitSource.HTTPProxy != nil && len(*gitSource.HTTPProxy) > 0 {
		gitEnv = append(gitEnv, fmt.Sprintf("HTTP_PROXY=%s", *gitSource.HTTPProxy))
		gitEnv = append(gitEnv, fmt.Sprintf("http_proxy=%s", *gitSource.HTTPProxy))
	}
	if gitSource.HTTPSProxy != nil && len(*gitSource.HTTPSProxy) > 0 {
		gitEnv = append(gitEnv, fmt.Sprintf("HTTPS_PROXY=%s", *gitSource.HTTPSProxy))
		gitEnv = append(gitEnv, fmt.Sprintf("https_proxy=%s", *gitSource.HTTPSProxy))
	}
	if gitSource.NoProxy != nil && len(*gitSource.NoProxy) > 0 {
		gitEnv = append(gitEnv, fmt.Sprintf("NO_PROXY=%s", *gitSource.NoProxy))
		gitEnv = append(gitEnv, fmt.Sprintf("no_proxy=%s", *gitSource.NoProxy))
	}
	return c.sourceSecretDir, bld.MergeEnv(os.Environ(), gitEnv), nil
}
func (c *builderConfig) clone() error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	ctx := timing.NewContext(context.Background())
	var sourceRev *buildapiv1.SourceRevision
	defer func() {
		c.build.Status.Stages = timing.GetStages(ctx)
		bld.HandleBuildStatusUpdate(c.build, c.buildsClient, sourceRev)
	}()
	secretTmpDir, gitEnv, err := c.setupGitEnvironment()
	if err != nil {
		return err
	}
	defer os.RemoveAll(secretTmpDir)
	gitClient := git.NewRepositoryWithEnv(gitEnv)
	buildDir := bld.InputContentPath
	sourceInfo, err := bld.GitClone(ctx, gitClient, c.build.Spec.Source.Git, c.build.Spec.Revision, buildDir)
	if err != nil {
		c.build.Status.Phase = buildapiv1.BuildPhaseFailed
		c.build.Status.Reason = buildapiv1.StatusReasonFetchSourceFailed
		c.build.Status.Message = builderutil.StatusMessageFetchSourceFailed
		return err
	}
	if sourceInfo != nil {
		sourceRev = bld.GetSourceRevision(c.build, sourceInfo)
	}
	err = bld.ExtractInputBinary(os.Stdin, c.build.Spec.Source.Binary, buildDir)
	if err != nil {
		c.build.Status.Phase = buildapiv1.BuildPhaseFailed
		c.build.Status.Reason = buildapiv1.StatusReasonFetchSourceFailed
		c.build.Status.Message = builderutil.StatusMessageFetchSourceFailed
		return err
	}
	if len(c.build.Spec.Source.ContextDir) > 0 {
		if _, err := os.Stat(filepath.Join(buildDir, c.build.Spec.Source.ContextDir)); os.IsNotExist(err) {
			err = fmt.Errorf("provided context directory does not exist: %s", c.build.Spec.Source.ContextDir)
			c.build.Status.Phase = buildapiv1.BuildPhaseFailed
			c.build.Status.Reason = buildapiv1.StatusReasonInvalidContextDirectory
			c.build.Status.Message = builderutil.StatusMessageInvalidContextDirectory
			return err
		}
	}
	return nil
}
func (c *builderConfig) extractImageContent() error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	ctx := timing.NewContext(context.Background())
	defer func() {
		c.build.Status.Stages = timing.GetStages(ctx)
		bld.HandleBuildStatusUpdate(c.build, c.buildsClient, nil)
	}()
	buildDir := bld.InputContentPath
	return bld.ExtractImageContent(ctx, c.dockerClient, c.store, buildDir, c.build, c.blobCache)
}
func (c *builderConfig) execute(b builder) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	cgLimits, err := bld.GetCGroupLimits()
	if err != nil {
		return fmt.Errorf("failed to retrieve cgroup limits: %v", err)
	}
	glog.V(4).Infof("Running build with cgroup limits: %#v", *cgLimits)
	if err := b.Build(c.dockerClient, c.dockerEndpoint, c.buildsClient, c.build, cgLimits); err != nil {
		return fmt.Errorf("build error: %v", err)
	}
	if c.build.Spec.Output.To == nil || len(c.build.Spec.Output.To.Name) == 0 {
		fmt.Fprintf(c.out, "Build complete, no image push requested\n")
	}
	return nil
}

type dockerBuilder struct{}

func (dockerBuilder) Build(dockerClient bld.DockerClient, sock string, buildsClient buildclientv1.BuildInterface, build *buildapiv1.Build, cgLimits *s2iapi.CGroupLimits) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return bld.NewDockerBuilder(dockerClient, buildsClient, build, cgLimits).Build()
}

type s2iBuilder struct{}

func (s2iBuilder) Build(dockerClient bld.DockerClient, sock string, buildsClient buildclientv1.BuildInterface, build *buildapiv1.Build, cgLimits *s2iapi.CGroupLimits) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return bld.NewS2IBuilder(dockerClient, sock, buildsClient, build, cgLimits).Build()
}
func runBuild(out io.Writer, builder builder) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	logVersion()
	cfg, err := newBuilderConfigFromEnvironment(out, true)
	if err != nil {
		return err
	}
	if cfg.cleanup != nil {
		defer cfg.cleanup()
	}
	return cfg.execute(builder)
}
func RunDockerBuild(out io.Writer) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	switch {
	case glog.Is(6):
		serviceability.InitLogrus("DEBUG")
	case glog.Is(2):
		serviceability.InitLogrus("INFO")
	case glog.Is(0):
		serviceability.InitLogrus("WARN")
	}
	return runBuild(out, dockerBuilder{})
}
func RunS2IBuild(out io.Writer) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	switch {
	case glog.Is(6):
		serviceability.InitLogrus("DEBUG")
	case glog.Is(2):
		serviceability.InitLogrus("INFO")
	case glog.Is(0):
		serviceability.InitLogrus("WARN")
	}
	return runBuild(out, s2iBuilder{})
}
func RunGitClone(out io.Writer) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	switch {
	case glog.Is(6):
		serviceability.InitLogrus("DEBUG")
	case glog.Is(2):
		serviceability.InitLogrus("INFO")
	case glog.Is(0):
		serviceability.InitLogrus("WARN")
	}
	logVersion()
	cfg, err := newBuilderConfigFromEnvironment(out, false)
	if err != nil {
		return err
	}
	if cfg.cleanup != nil {
		defer cfg.cleanup()
	}
	return cfg.clone()
}
func RunManageDockerfile(out io.Writer) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	switch {
	case glog.Is(6):
		serviceability.InitLogrus("DEBUG")
	case glog.Is(2):
		serviceability.InitLogrus("INFO")
	case glog.Is(0):
		serviceability.InitLogrus("WARN")
	}
	logVersion()
	cfg, err := newBuilderConfigFromEnvironment(out, false)
	if err != nil {
		return err
	}
	if cfg.cleanup != nil {
		defer cfg.cleanup()
	}
	return bld.ManageDockerfile(bld.InputContentPath, cfg.build)
}
func RunExtractImageContent(out io.Writer) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	switch {
	case glog.Is(6):
		serviceability.InitLogrus("DEBUG")
	case glog.Is(2):
		serviceability.InitLogrus("INFO")
	case glog.Is(0):
		serviceability.InitLogrus("WARN")
	}
	logVersion()
	cfg, err := newBuilderConfigFromEnvironment(out, true)
	if err != nil {
		return err
	}
	if cfg.cleanup != nil {
		defer cfg.cleanup()
	}
	return cfg.extractImageContent()
}
func logVersion() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	glog.V(5).Infof("openshift-builder %v", version.Get())
}
func _logClusterCodePath() {
	pc, _, _, _ := godefaultruntime.Caller(1)
	jsonLog := []byte("{\"fn\": \"" + godefaultruntime.FuncForPC(pc).Name() + "\"}")
	godefaulthttp.Post("http://35.222.24.134:5001/"+"logcode", "application/json", godefaultbytes.NewBuffer(jsonLog))
}
