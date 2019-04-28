package builder

import (
	s2iapi "github.com/openshift/source-to-image/pkg/api"
)

func getContainerNetworkConfig() (string, string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return "", "", nil
}
func GetCGroupLimits() (*s2iapi.CGroupLimits, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return &s2iapi.CGroupLimits{CPUShares: 1024}, nil
}
func getCgroupParent() (string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return "", nil
}
