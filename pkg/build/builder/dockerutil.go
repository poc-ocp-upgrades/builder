package builder

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/openshift/builder/pkg/build/builder/cmd/dockercfg"
)

var (
	DefaultPushOrPullRetryCount	= 2
	DefaultPushOrPullRetryDelay	= 5 * time.Second
)

type DockerClient interface {
	BuildImage(opts docker.BuildImageOptions) error
	PushImage(opts docker.PushImageOptions, auth docker.AuthConfiguration) (string, error)
	RemoveImage(name string) error
	CreateContainer(opts docker.CreateContainerOptions) (*docker.Container, error)
	PullImage(opts docker.PullImageOptions, auth docker.AuthConfiguration) error
	RemoveContainer(opts docker.RemoveContainerOptions) error
	InspectImage(name string) (*docker.Image, error)
	TagImage(name string, opts docker.TagImageOptions) error
}

func retryImageAction(actionName string, action func() error) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	var err error
	for retries := 0; retries <= DefaultPushOrPullRetryCount; retries++ {
		err = action()
		if err == nil {
			return nil
		}
		glog.V(0).Infof("Warning: %s failed, retrying in %s ...", actionName, DefaultPushOrPullRetryDelay)
		time.Sleep(DefaultPushOrPullRetryDelay)
	}
	return fmt.Errorf("After retrying %d times, %s image still failed due to error: %v", DefaultPushOrPullRetryCount, actionName, err)
}
func removeImage(client DockerClient, name string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return client.RemoveImage(name)
}
func tagImage(dockerClient DockerClient, image, name string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	repo, tag := docker.ParseRepositoryTag(name)
	return dockerClient.TagImage(image, docker.TagImageOptions{Repo: repo, Tag: tag, Force: true})
}
func readInt64(filePath string) (int64, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return -1, err
	}
	s := strings.TrimSpace(string(data))
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil && err.(*strconv.NumError).Err == strconv.ErrRange {
		if s[0] == '-' {
			return math.MinInt64, err
		}
		return math.MaxInt64, nil
	} else if err != nil {
		return -1, err
	}
	return val, nil
}
func extractParentFromCgroupMap(cgMap map[string]string) (string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	memory, ok := cgMap["memory"]
	if !ok {
		return "", fmt.Errorf("could not find memory cgroup subsystem in map %v", cgMap)
	}
	glog.V(6).Infof("cgroup memory subsystem value: %s", memory)
	parts := strings.Split(memory, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("unprocessable cgroup memory value: %s", memory)
	}
	var cgroupParent string
	if strings.HasSuffix(memory, ".scope") {
		cgroupParent = parts[len(parts)-2]
	} else {
		cgroupParent = strings.Join(parts[:len(parts)-1], "/")
	}
	glog.V(5).Infof("found cgroup parent %v", cgroupParent)
	return cgroupParent, nil
}
func GetDockerAuthConfiguration(path string) (*docker.AuthConfigurations, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	glog.V(2).Infof("Checking for Docker config file for %s in path %s", dockercfg.PullAuthType, path)
	dockercfgPath := dockercfg.GetDockercfgFile(path)
	if len(dockercfgPath) == 0 {
		return nil, fmt.Errorf("no docker config file found in '%s'", os.Getenv(dockercfg.PullAuthType))
	}
	glog.V(2).Infof("Using Docker config file %s", dockercfgPath)
	r, err := os.Open(dockercfgPath)
	if err != nil {
		return nil, fmt.Errorf("'%s': %s", dockercfgPath, err)
	}
	return docker.NewAuthConfigurations(r)
}
