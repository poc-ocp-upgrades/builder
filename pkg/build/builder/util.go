package builder

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	docker "github.com/fsouza/go-dockerclient"
	s2iapi "github.com/openshift/source-to-image/pkg/api"
	s2iutil "github.com/openshift/source-to-image/pkg/util"
	buildapiv1 "github.com/openshift/api/build/v1"
	builderutil "github.com/openshift/builder/pkg/build/builder/util"
)

const (
	ConfigMapCertsMountPath	= "/var/run/configs/openshift.io/certs"
	SecretCertsMountPath	= "/var/run/secrets/kubernetes.io/serviceaccount"
)

var (
	procCGroupPattern	= regexp.MustCompile(`\d+:([a-z_,]+):/.*/(\w+-|)([a-z0-9]+).*`)
	ClientTypeUnknown	= errors.New("internal error: method not implemented for this client type")
)

func MergeEnv(oldEnv, newEnv []string) []string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	key := func(e string) string {
		i := strings.Index(e, "=")
		if i == -1 {
			return e
		}
		return e[:i]
	}
	result := []string{}
	newVars := map[string]struct{}{}
	for _, e := range newEnv {
		newVars[key(e)] = struct{}{}
	}
	result = append(result, newEnv...)
	for _, e := range oldEnv {
		if _, exists := newVars[key(e)]; exists {
			continue
		}
		result = append(result, e)
	}
	return result
}
func reportPushFailure(err error, authPresent bool, pushAuthConfig docker.AuthConfiguration) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if authPresent {
		glog.V(0).Infof("Registry server Address: %s", pushAuthConfig.ServerAddress)
		glog.V(0).Infof("Registry server User Name: %s", pushAuthConfig.Username)
		glog.V(0).Infof("Registry server Email: %s", pushAuthConfig.Email)
		passwordPresent := "<<empty>>"
		if len(pushAuthConfig.Password) > 0 {
			passwordPresent = "<<non-empty>>"
		}
		glog.V(0).Infof("Registry server Password: %s", passwordPresent)
	}
	return fmt.Errorf("Failed to push image: %v", err)
}
func addBuildLabels(labels map[string]string, build *buildapiv1.Build) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	labels[builderutil.DefaultDockerLabelNamespace+"build.name"] = build.Name
	labels[builderutil.DefaultDockerLabelNamespace+"build.namespace"] = build.Namespace
}
func SafeForLoggingEnvironmentList(env s2iapi.EnvironmentList) s2iapi.EnvironmentList {
	_logClusterCodePath()
	defer _logClusterCodePath()
	newEnv := make(s2iapi.EnvironmentList, len(env))
	copy(newEnv, env)
	proxyRegex := regexp.MustCompile("(?i)proxy")
	for i, env := range newEnv {
		if proxyRegex.MatchString(env.Name) {
			newEnv[i].Value, _ = s2iutil.SafeForLoggingURL(env.Value)
		}
	}
	return newEnv
}
func SafeForLoggingS2IConfig(config *s2iapi.Config) *s2iapi.Config {
	_logClusterCodePath()
	defer _logClusterCodePath()
	newConfig := *config
	newConfig.Environment = SafeForLoggingEnvironmentList(config.Environment)
	if config.ScriptDownloadProxyConfig != nil {
		newProxy := *config.ScriptDownloadProxyConfig
		newConfig.ScriptDownloadProxyConfig = &newProxy
		if newConfig.ScriptDownloadProxyConfig.HTTPProxy != nil {
			newConfig.ScriptDownloadProxyConfig.HTTPProxy = builderutil.SafeForLoggingURL(newConfig.ScriptDownloadProxyConfig.HTTPProxy)
		}
		if newConfig.ScriptDownloadProxyConfig.HTTPProxy != nil {
			newConfig.ScriptDownloadProxyConfig.HTTPSProxy = builderutil.SafeForLoggingURL(newConfig.ScriptDownloadProxyConfig.HTTPProxy)
		}
	}
	newConfig.ScriptsURL, _ = s2iutil.SafeForLoggingURL(newConfig.ScriptsURL)
	return &newConfig
}
func ReadLines(fileName string) ([]string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}
func ParseProxyURL(proxy string) (*url.URL, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	proxyURL, err := url.Parse(proxy)
	if err != nil || !strings.HasPrefix(proxyURL.Scheme, "http") {
		if proxyURL, err := url.Parse("http://" + proxy); err == nil {
			return proxyURL, nil
		}
	}
	return proxyURL, err
}
