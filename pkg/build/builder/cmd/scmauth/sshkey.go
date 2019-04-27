package scmauth

import (
	"io/ioutil"
	"path/filepath"
)

const SSHPrivateKeyMethodName = "ssh-privatekey"

type SSHPrivateKey struct{}

func (_ SSHPrivateKey) Setup(baseDir string, context SCMAuthContext) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	script, err := ioutil.TempFile("", "gitssh")
	if err != nil {
		return err
	}
	defer script.Close()
	if err := script.Chmod(0711); err != nil {
		return err
	}
	content := "#!/bin/sh\nssh -i " + filepath.Join(baseDir, SSHPrivateKeyMethodName) + " -o StrictHostKeyChecking=false \"$@\"\n"
	glog.V(5).Infof("Adding Private SSH Auth:\n%s\n", content)
	if _, err := script.WriteString(content); err != nil {
		return err
	}
	if err := context.Set("GIT_SSH", script.Name()); err != nil {
		return err
	}
	return nil
}
func (_ SSHPrivateKey) Name() string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return SSHPrivateKeyMethodName
}
func (_ SSHPrivateKey) Handles(name string) bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return name == SSHPrivateKeyMethodName
}
