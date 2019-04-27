package builder

import (
	"fmt"
	"os"
	"github.com/opencontainers/runc/libcontainer/cgroups"
	s2iapi "github.com/openshift/source-to-image/pkg/api"
)

func GetCGroupLimits() (*s2iapi.CGroupLimits, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	byteLimit, err := readInt64("/sys/fs/cgroup/memory/memory.limit_in_bytes")
	if err != nil {
		if _, err := os.Stat("/sys/fs/cgroup"); os.IsNotExist(err) {
			return &s2iapi.CGroupLimits{}, nil
		}
		return nil, fmt.Errorf("cannot determine cgroup limits: %v", err)
	}
	if byteLimit > 92233720368547 {
		byteLimit = 92233720368547
	}
	parent, err := getCgroupParent()
	if err != nil {
		return nil, fmt.Errorf("read cgroup parent: %v", err)
	}
	return &s2iapi.CGroupLimits{MemoryLimitBytes: byteLimit, MemorySwap: byteLimit, Parent: parent}, nil
}
func getCgroupParent() (string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	cgMap, err := cgroups.ParseCgroupFile("/proc/self/cgroup")
	if err != nil {
		return "", err
	}
	glog.V(6).Infof("found cgroup values map: %v", cgMap)
	return extractParentFromCgroupMap(cgMap)
}
