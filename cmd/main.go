package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"time"
	"github.com/containers/storage/pkg/reexec"
	"github.com/spf13/cobra"
	"k8s.io/apiserver/pkg/util/logs"
	"github.com/openshift/builder/pkg/build/builder"
	"github.com/openshift/builder/pkg/version"
	"github.com/openshift/library-go/pkg/serviceability"
)

func main() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if reexec.Init() {
		return
	}
	logs.InitLogs()
	defer logs.FlushLogs()
	defer serviceability.BehaviorOnPanic(os.Getenv("OPENSHIFT_ON_PANIC"), version.Get())()
	defer serviceability.Profile(os.Getenv("OPENSHIFT_PROFILE")).Stop()
	rand.Seed(time.Now().UTC().UnixNano())
	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}
	const tlsCertRoot = "/etc/pki/tls/certs"
	const runtimeCertRoot = "/etc/docker/certs.d"
	clusterCASrc := fmt.Sprintf("%s/ca.crt", builder.SecretCertsMountPath)
	clusterCADst := fmt.Sprintf("%s/cluster.crt", tlsCertRoot)
	err := CopyFileIfExists(clusterCASrc, clusterCADst)
	if err != nil {
		fmt.Printf("Error setting up cluster CA cert: %v", err)
		os.Exit(1)
	}
	oldServiceCASrc := fmt.Sprintf("%s/service-ca.crt", builder.SecretCertsMountPath)
	oldServiceCADst := fmt.Sprintf("%s/service.crt", tlsCertRoot)
	err = CopyFileIfExists(oldServiceCASrc, oldServiceCADst)
	if err != nil {
		fmt.Printf("Error setting up service CA cert: %v", err)
		os.Exit(1)
	}
	newServiceCASrc := fmt.Sprintf("%s/service-ca.crt", builder.ConfigMapCertsMountPath)
	newServiceCADst := fmt.Sprintf("%s/openshift-service.crt", tlsCertRoot)
	err = CopyFileIfExists(newServiceCASrc, newServiceCADst)
	if err != nil {
		fmt.Printf("Error setting up service CA cert: %v", err)
		os.Exit(1)
	}
	runtimeCASrc := fmt.Sprintf("%s/certs.d", builder.ConfigMapCertsMountPath)
	err = CopyDirIfExists(runtimeCASrc, runtimeCertRoot)
	if err != nil {
		fmt.Printf("Error setting up service CA cert: %v", err)
		os.Exit(1)
	}
	basename := filepath.Base(os.Args[0])
	command := CommandFor(basename)
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
func CopyDirIfExists(src, dst string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	srcInfo, err := os.Stat(src)
	if os.IsNotExist(err) {
		return nil
	}
	if err = os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}
	dirInfo, err := ioutil.ReadDir(src)
	for _, info := range dirInfo {
		srcPath := filepath.Join(src, info.Name())
		dstPath := filepath.Join(dst, info.Name())
		if info.IsDir() {
			err = CopyDirIfExists(srcPath, dstPath)
		} else {
			err = CopyFileIfExists(srcPath, dstPath)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
func CopyFileIfExists(src, dst string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_, err := os.Stat(src)
	if os.IsNotExist(err) {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}
func CommandFor(basename string) *cobra.Command {
	_logClusterCodePath()
	defer _logClusterCodePath()
	var cmd *cobra.Command
	switch basename {
	case "openshift-sti-build":
		cmd = NewCommandS2IBuilder(basename)
	case "openshift-docker-build":
		cmd = NewCommandDockerBuilder(basename)
	case "openshift-git-clone":
		cmd = NewCommandGitClone(basename)
	case "openshift-manage-dockerfile":
		cmd = NewCommandManageDockerfile(basename)
	case "openshift-extract-image-content":
		cmd = NewCommandExtractImageContent(basename)
	default:
		fmt.Printf("unknown command name: %s\n", basename)
		os.Exit(1)
	}
	GLog(cmd.PersistentFlags())
	return cmd
}
