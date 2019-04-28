package scmauth

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	builder "github.com/openshift/builder/pkg/build/builder"
	utilglog "github.com/openshift/builder/pkg/build/builder/util/glog"
)

var glog = utilglog.ToFile(os.Stderr, 2)

func createGitConfig(includePath string, context SCMAuthContext) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	tempDir, err := ioutil.TempDir("", "git")
	if err != nil {
		return err
	}
	gitconfig := filepath.Join(tempDir, ".gitconfig")
	content := fmt.Sprintf("[include]\npath = %s\n", includePath)
	if err := ioutil.WriteFile(gitconfig, []byte(content), 0600); err != nil {
		return err
	}
	if err := context.Set("HOME", tempDir); err != nil {
		return err
	}
	if err := context.Set("GIT_CONFIG", gitconfig); err != nil {
		return err
	}
	return nil
}
func ensureGitConfigIncludes(path string, context SCMAuthContext) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	gitconfig, present := context.Get("GIT_CONFIG")
	if !present {
		return createGitConfig(path, context)
	}
	lines, err := builder.ReadLines(gitconfig)
	if err != nil {
		return err
	}
	for _, line := range lines {
		if line == fmt.Sprintf("path = %s", path) {
			return nil
		}
	}
	lines = append(lines, fmt.Sprintf("path = %s", path))
	content := []byte(strings.Join(lines, "\n"))
	return ioutil.WriteFile(gitconfig, content, 0600)
}
