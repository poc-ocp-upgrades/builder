package scmauth

import (
	"path/filepath"
)

const GitConfigName = ".gitconfig"

type GitConfig struct{}

func (_ GitConfig) Setup(baseDir string, context SCMAuthContext) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	glog.V(4).Infof("Adding user-provided gitconfig %s to build gitconfig", filepath.Join(baseDir, GitConfigName))
	return ensureGitConfigIncludes(filepath.Join(baseDir, GitConfigName), context)
}
func (_ GitConfig) Name() string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return GitConfigName
}
func (_ GitConfig) Handles(name string) bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return name == GitConfigName
}
