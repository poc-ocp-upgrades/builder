package builder

import (
	"errors"
	s2iapi "github.com/openshift/source-to-image/pkg/api"
)

func getContainerNetworkConfig() (string, string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return "", "", errors.New("getContainerNetworkConfig is unsupported on this platform")
}
func GetCGroupLimits() (*s2iapi.CGroupLimits, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return nil, errors.New("GetCGroupLimits is unsupported on this platform")
}
func getCgroupParent() (string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return "", errors.New("getCgroupParent is unsupported on this platform")
}
