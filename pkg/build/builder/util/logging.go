package util

import (
	"net/url"
	"regexp"
	s2iapi "github.com/openshift/source-to-image/pkg/api"
	s2iutil "github.com/openshift/source-to-image/pkg/util"
	buildapiv1 "github.com/openshift/api/build/v1"
	corev1 "k8s.io/api/core/v1"
)

var (
	proxyRegex = regexp.MustCompile("(?i)proxy")
)

func SafeForLoggingURL(u *url.URL) *url.URL {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	if u == nil {
		return nil
	}
	newURL, err := url.Parse(u.String())
	if err != nil {
		return nil
	}
	if newURL.User != nil {
		if _, passwordSet := newURL.User.Password(); passwordSet {
			newURL.User = url.User("redacted")
		}
	}
	return newURL
}
func SafeForLoggingEnvVar(env []corev1.EnvVar) []corev1.EnvVar {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	newEnv := make([]corev1.EnvVar, len(env))
	copy(newEnv, env)
	for i, env := range newEnv {
		if proxyRegex.MatchString(env.Name) {
			newEnv[i].Value, _ = s2iutil.SafeForLoggingURL(env.Value)
		}
	}
	return newEnv
}
func SafeForLoggingBuildCommonSpec(spec *buildapiv1.CommonSpec) *buildapiv1.CommonSpec {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	newSpec := spec.DeepCopy()
	if newSpec.Source.Git != nil {
		if newSpec.Source.Git.HTTPProxy != nil {
			s, _ := s2iutil.SafeForLoggingURL(*newSpec.Source.Git.HTTPProxy)
			newSpec.Source.Git.HTTPProxy = &s
		}
		if newSpec.Source.Git.HTTPSProxy != nil {
			s, _ := s2iutil.SafeForLoggingURL(*newSpec.Source.Git.HTTPSProxy)
			newSpec.Source.Git.HTTPSProxy = &s
		}
	}
	if newSpec.Strategy.SourceStrategy != nil {
		newSpec.Strategy.SourceStrategy.Env = SafeForLoggingEnvVar(newSpec.Strategy.SourceStrategy.Env)
	}
	if newSpec.Strategy.DockerStrategy != nil {
		newSpec.Strategy.DockerStrategy.Env = SafeForLoggingEnvVar(newSpec.Strategy.DockerStrategy.Env)
	}
	if newSpec.Strategy.CustomStrategy != nil {
		newSpec.Strategy.CustomStrategy.Env = SafeForLoggingEnvVar(newSpec.Strategy.CustomStrategy.Env)
	}
	if newSpec.Strategy.JenkinsPipelineStrategy != nil {
		newSpec.Strategy.JenkinsPipelineStrategy.Env = SafeForLoggingEnvVar(newSpec.Strategy.JenkinsPipelineStrategy.Env)
	}
	return newSpec
}
func SafeForLoggingBuild(build *buildapiv1.Build) *buildapiv1.Build {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	newBuild := *build
	newSpec := SafeForLoggingBuildCommonSpec(&build.Spec.CommonSpec)
	newBuild.Spec.CommonSpec = *newSpec
	return &newBuild
}
func SafeForLoggingEnvironmentList(env s2iapi.EnvironmentList) s2iapi.EnvironmentList {
	_logClusterCodePath()
	defer _logClusterCodePath()
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
	_logClusterCodePath()
	defer _logClusterCodePath()
	newConfig := *config
	newConfig.Environment = SafeForLoggingEnvironmentList(config.Environment)
	if config.ScriptDownloadProxyConfig != nil {
		newProxy := *config.ScriptDownloadProxyConfig
		newConfig.ScriptDownloadProxyConfig = &newProxy
		if newConfig.ScriptDownloadProxyConfig.HTTPProxy != nil {
			newConfig.ScriptDownloadProxyConfig.HTTPProxy = SafeForLoggingURL(newConfig.ScriptDownloadProxyConfig.HTTPProxy)
		}
		if newConfig.ScriptDownloadProxyConfig.HTTPProxy != nil {
			newConfig.ScriptDownloadProxyConfig.HTTPSProxy = SafeForLoggingURL(newConfig.ScriptDownloadProxyConfig.HTTPProxy)
		}
	}
	newConfig.ScriptsURL, _ = s2iutil.SafeForLoggingURL(newConfig.ScriptsURL)
	return &newConfig
}
