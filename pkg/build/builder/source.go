package builder

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	realglog "github.com/golang/glog"
	"github.com/containers/image/pkg/docker/config"
	"github.com/containers/image/types"
	"github.com/containers/storage"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/containers/buildah"
	s2igit "github.com/openshift/source-to-image/pkg/scm/git"
	buildapiv1 "github.com/openshift/api/build/v1"
	"github.com/openshift/builder/pkg/build/builder/cmd/dockercfg"
	"github.com/openshift/builder/pkg/build/builder/timing"
	"github.com/openshift/library-go/pkg/git"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	initialURLCheckTimeout	= 16 * time.Second
	timeoutIncrementFactor	= 4
)

type gitAuthError string
type gitNotFoundError string

func (e gitAuthError) Error() string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return fmt.Sprintf("failed to fetch requested repository %q with provided credentials", string(e))
}
func (e gitNotFoundError) Error() string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return fmt.Sprintf("requested repository %q not found", string(e))
}
func GitClone(ctx context.Context, gitClient GitClient, gitSource *buildapiv1.GitBuildSource, revision *buildapiv1.SourceRevision, dir string) (*git.SourceInfo, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	err := os.RemoveAll(dir)
	if err != nil {
		return nil, err
	}
	os.MkdirAll(dir, 0777)
	hasGitSource, err := extractGitSource(ctx, gitClient, gitSource, revision, dir, initialURLCheckTimeout)
	if err != nil {
		return nil, err
	}
	var sourceInfo *git.SourceInfo
	if hasGitSource {
		var errs []error
		sourceInfo, errs = gitClient.GetInfo(dir)
		if len(errs) > 0 {
			for _, e := range errs {
				glog.V(0).Infof("error: Unable to retrieve Git info: %v", e)
			}
		}
		if sourceInfo != nil {
			sourceInfoJson, err := json.Marshal(*sourceInfo)
			if err != nil {
				glog.V(0).Infof("error: Unable to serialized git source info: %v", err)
				return sourceInfo, nil
			}
			err = ioutil.WriteFile(filepath.Join(buildWorkDirMount, "sourceinfo.json"), sourceInfoJson, 0644)
			if err != nil {
				glog.V(0).Infof("error: Unable to serialized git source info: %v", err)
				return sourceInfo, nil
			}
		}
	}
	return sourceInfo, nil
}
func ManageDockerfile(dir string, build *buildapiv1.Build) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	os.MkdirAll(dir, 0777)
	glog.V(5).Infof("Checking for presence of a Dockerfile")
	if dockerfileSource := build.Spec.Source.Dockerfile; dockerfileSource != nil {
		baseDir := dir
		if len(build.Spec.Source.ContextDir) != 0 {
			baseDir = filepath.Join(baseDir, build.Spec.Source.ContextDir)
		}
		if err := ioutil.WriteFile(filepath.Join(baseDir, "Dockerfile"), []byte(*dockerfileSource), 0660); err != nil {
			return err
		}
	}
	if build.Spec.Strategy.DockerStrategy != nil {
		sourceInfo, err := readSourceInfo()
		if err != nil {
			return fmt.Errorf("error reading git source info: %v", err)
		}
		return addBuildParameters(dir, build, sourceInfo)
	}
	return nil
}
func ExtractImageContent(ctx context.Context, dockerClient DockerClient, store storage.Store, dir string, build *buildapiv1.Build, blobCacheDirectory string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	os.MkdirAll(dir, 0777)
	forcePull := false
	switch {
	case build.Spec.Strategy.SourceStrategy != nil:
		forcePull = build.Spec.Strategy.SourceStrategy.ForcePull
	case build.Spec.Strategy.DockerStrategy != nil:
		forcePull = build.Spec.Strategy.DockerStrategy.ForcePull
	case build.Spec.Strategy.CustomStrategy != nil:
		forcePull = build.Spec.Strategy.CustomStrategy.ForcePull
	}
	for i, image := range build.Spec.Source.Images {
		if len(image.Paths) == 0 {
			continue
		}
		imageSecretIndex := i
		if image.PullSecret == nil {
			imageSecretIndex = -1
		}
		err := extractSourceFromImage(ctx, dockerClient, store, image.From.Name, dir, imageSecretIndex, image.Paths, forcePull, blobCacheDirectory)
		if err != nil {
			return err
		}
	}
	return nil
}
func checkRemoteGit(gitClient GitClient, url string, initialTimeout time.Duration) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	var (
		out	string
		errOut	string
		err	error
	)
	timeout := initialTimeout
	for {
		glog.V(4).Infof("git ls-remote --heads %s", url)
		out, errOut, err = gitClient.TimedListRemote(timeout, url, "--heads")
		if len(out) != 0 {
			glog.V(4).Infof(out)
		}
		if len(errOut) != 0 {
			glog.V(4).Infof(errOut)
		}
		if err != nil {
			if _, ok := err.(*git.TimeoutError); ok {
				timeout = timeout * timeoutIncrementFactor
				glog.Infof("WARNING: timed out waiting for git server, will wait %s", timeout)
				continue
			}
		}
		break
	}
	if err != nil {
		combinedOut := out + errOut
		switch {
		case strings.Contains(combinedOut, "Authentication failed"):
			return gitAuthError(url)
		case strings.Contains(combinedOut, "not found"):
			return gitNotFoundError(url)
		}
	}
	return err
}
func checkSourceURI(gitClient GitClient, rawurl string, timeout time.Duration) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_, err := s2igit.Parse(rawurl)
	if err != nil {
		return fmt.Errorf("Invalid git source url %q: %v", rawurl, err)
	}
	return checkRemoteGit(gitClient, rawurl, timeout)
}
func ExtractInputBinary(in io.Reader, source *buildapiv1.BinaryBuildSource, dir string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	os.MkdirAll(dir, 0777)
	if source == nil {
		return nil
	}
	var path string
	if len(source.AsFile) > 0 {
		glog.V(0).Infof("Receiving source from STDIN as file %s", source.AsFile)
		path = filepath.Join(dir, source.AsFile)
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0664)
		if err != nil {
			return err
		}
		defer f.Close()
		n, err := io.Copy(f, os.Stdin)
		if err != nil {
			return err
		}
		glog.V(4).Infof("Received %d bytes into %s", n, path)
		return nil
	}
	glog.V(0).Infof("Receiving source from STDIN as archive ...")
	args := []string{"-x", "-o", "-m", "-f", "-", "-C", dir}
	if glog.Is(6) {
		args = append(args, "-v")
	}
	cmd := exec.Command("bsdtar", args...)
	cmd.Stdin = in
	out, err := cmd.CombinedOutput()
	glog.V(4).Infof("Extracting...\n%s", string(out))
	if err != nil {
		return fmt.Errorf("unable to extract binary build input, must be a zip, tar, or gzipped tar, or specified as a file: %v", err)
	}
	return nil
}
func extractGitSource(ctx context.Context, gitClient GitClient, gitSource *buildapiv1.GitBuildSource, revision *buildapiv1.SourceRevision, dir string, timeout time.Duration) (bool, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if gitSource == nil {
		return false, nil
	}
	glog.V(0).Infof("Cloning %q ...", gitSource.URI)
	if err := checkSourceURI(gitClient, gitSource.URI, timeout); err != nil {
		return true, err
	}
	cloneOptions := []string{}
	usingRevision := revision != nil && revision.Git != nil && len(revision.Git.Commit) != 0
	usingRef := len(gitSource.Ref) != 0 || usingRevision
	if !usingRef {
		cloneOptions = append(cloneOptions, "--recursive")
		cloneOptions = append(cloneOptions, git.Shallow)
	}
	glog.V(3).Infof("Cloning source from %s", gitSource.URI)
	if !glog.Is(5) {
		cloneOptions = append(cloneOptions, "--quiet")
	}
	startTime := metav1.Now()
	if err := gitClient.CloneWithOptions(dir, gitSource.URI, cloneOptions...); err != nil {
		return true, err
	}
	timing.RecordNewStep(ctx, buildapiv1.StageFetchInputs, buildapiv1.StepFetchGitSource, startTime, metav1.Now())
	if usingRef {
		commit := gitSource.Ref
		if usingRevision {
			commit = revision.Git.Commit
		}
		if err := gitClient.Checkout(dir, commit); err != nil {
			err = gitClient.PotentialPRRetryAsFetch(dir, gitSource.URI, commit, err)
			if err != nil {
				return true, err
			}
		}
		if err := gitClient.SubmoduleUpdate(dir, true, true); err != nil {
			return true, err
		}
	}
	if information, gitErr := gitClient.GetInfo(dir); len(gitErr) == 0 {
		glog.Infof("\tCommit:\t%s (%s)\n", information.CommitID, information.Message)
		glog.Infof("\tAuthor:\t%s <%s>\n", information.AuthorName, information.AuthorEmail)
		glog.Infof("\tDate:\t%s\n", information.Date)
	}
	return true, nil
}
func copyImageSourceFromFilesytem(sourceDir, destDir string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	fi, err := os.Stat(destDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		glog.V(4).Infof("Creating image destination directory: %s", destDir)
		err := os.MkdirAll(destDir, 0755)
		if err != nil {
			return err
		}
	} else {
		if !fi.IsDir() {
			return fmt.Errorf("destination %s must be a directory", destDir)
		}
	}
	args := []string{"-r"}
	if glog.Is(5) {
		args = append(args, "-v")
	}
	args = append(args, sourceDir, destDir)
	out, err := exec.Command("cp", args...).CombinedOutput()
	glog.V(4).Infof("copying image content: %s", string(out))
	if err != nil {
		return err
	}
	return nil
}
func extractSourceFromImage(ctx context.Context, dockerClient DockerClient, store storage.Store, image, buildDir string, imageSecretIndex int, paths []buildapiv1.ImageSourcePath, forcePull bool, blobCacheDirectory string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	glog.V(4).Infof("Extracting image source from image %s", image)
	pullPolicy := buildah.PullIfMissing
	if forcePull {
		pullPolicy = buildah.PullAlways
	}
	var auths *docker.AuthConfigurations
	var err error
	if imageSecretIndex != -1 {
		pullSecretPath := os.Getenv(fmt.Sprintf("%s%d", dockercfg.PullSourceAuthType, imageSecretIndex))
		if len(pullSecretPath) > 0 {
			auths, err = GetDockerAuthConfiguration(pullSecretPath)
			if err != nil {
				return fmt.Errorf("error reading docker auth configuration: %v", err)
			}
		}
	}
	var systemContext types.SystemContext
	systemContext.AuthFilePath = "/tmp/config.json"
	systemContext.OCIInsecureSkipTLSVerify = true
	systemContext.DockerInsecureSkipTLSVerify = types.NewOptionalBool(true)
	if auths != nil {
		for registry, ac := range auths.Configs {
			glog.V(5).Infof("Setting authentication for registry %q using %q.", registry, ac.ServerAddress)
			if err := config.SetAuthentication(&systemContext, registry, ac.Username, ac.Password); err != nil {
				return err
			}
			if err := config.SetAuthentication(&systemContext, ac.ServerAddress, ac.Username, ac.Password); err != nil {
				return err
			}
		}
	}
	builderOptions := buildah.BuilderOptions{FromImage: image, PullPolicy: pullPolicy, ReportWriter: os.Stdout, SystemContext: &systemContext, CommonBuildOpts: &buildah.CommonBuildOptions{}}
	builder, err := buildah.NewBuilder(ctx, store, builderOptions)
	if err != nil {
		return fmt.Errorf("error creating buildah builder: %v", err)
	}
	mountPath, err := builder.Mount("")
	defer func() {
		err := builder.Unmount()
		if err != nil {
			realglog.Errorf("failed to unmount: %v", err)
		}
	}()
	if err != nil {
		return fmt.Errorf("error mounting image content from image %s: %v", image, err)
	}
	for _, path := range paths {
		destPath := filepath.Join(buildDir, path.DestinationDir)
		sourcePath := filepath.Join(mountPath, path.SourcePath)
		if strings.HasSuffix(path.SourcePath, "/.") {
			sourcePath = sourcePath + "/."
		}
		glog.V(4).Infof("Extracting path %s from image %s to %s", path.SourcePath, image, path.DestinationDir)
		err := copyImageSourceFromFilesytem(sourcePath, destPath)
		if err != nil {
			return fmt.Errorf("error copying source path %s to %s: %v", path.SourcePath, path.DestinationDir, err)
		}
	}
	return nil
}
