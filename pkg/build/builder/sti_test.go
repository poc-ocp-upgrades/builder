package builder

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	docker "github.com/fsouza/go-dockerclient"
	buildapiv1 "github.com/openshift/api/build/v1"
	buildfake "github.com/openshift/client-go/build/clientset/versioned/fake"
	"github.com/openshift/library-go/pkg/git"
	s2iapi "github.com/openshift/source-to-image/pkg/api"
	s2iconstants "github.com/openshift/source-to-image/pkg/api/constants"
	s2ibuild "github.com/openshift/source-to-image/pkg/build"
)

type testStiBuilderFactory struct {
	getStrategyErr	error
	buildError	error
}

func (factory testStiBuilderFactory) Builder(config *s2iapi.Config, overrides s2ibuild.Overrides) (s2ibuild.Builder, s2iapi.BuildInfo, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if factory.getStrategyErr != nil {
		return nil, s2iapi.BuildInfo{}, factory.getStrategyErr
	}
	return testBuilder{buildError: factory.buildError}, s2iapi.BuildInfo{}, nil
}

type testBuilder struct{ buildError error }

func (builder testBuilder) Build(config *s2iapi.Config) (*s2iapi.Result, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return &s2iapi.Result{BuildInfo: s2iapi.BuildInfo{}}, builder.buildError
}

type testS2IBuilderConfig struct {
	pullImageFunc		func(opts docker.PullImageOptions, auth docker.AuthConfiguration) error
	inspectImageFunc	func(name string) (*docker.Image, error)
	errPushImage		error
	getStrategyErr		error
	buildError		error
}

func newTestS2IBuilder(config testS2IBuilderConfig) *S2IBuilder {
	_logClusterCodePath()
	defer _logClusterCodePath()
	client := &buildfake.Clientset{}
	return newS2IBuilder(&FakeDocker{pullImageFunc: config.pullImageFunc, inspectImageFunc: config.inspectImageFunc, errPushImage: config.errPushImage}, "unix:///var/run/docker2.sock", client.Build().Builds(""), makeBuild(), testStiBuilderFactory{getStrategyErr: config.getStrategyErr, buildError: config.buildError}, runtimeConfigValidator{}, nil)
}
func makeBuild() *buildapiv1.Build {
	_logClusterCodePath()
	defer _logClusterCodePath()
	t := true
	return &buildapiv1.Build{ObjectMeta: metav1.ObjectMeta{Name: "build-1", Namespace: "ns"}, Spec: buildapiv1.BuildSpec{CommonSpec: buildapiv1.CommonSpec{Source: buildapiv1.BuildSource{}, Strategy: buildapiv1.BuildStrategy{SourceStrategy: &buildapiv1.SourceBuildStrategy{Env: append([]corev1.EnvVar{}, corev1.EnvVar{Name: "HTTPS_PROXY", Value: "https://test/secure:8443"}, corev1.EnvVar{Name: "HTTP_PROXY", Value: "http://test/insecure:8080"}), From: corev1.ObjectReference{Kind: "DockerImage", Name: "test/builder:latest"}, Incremental: &t}}, Output: buildapiv1.BuildOutput{To: &corev1.ObjectReference{Kind: "DockerImage", Name: "test/test-result:latest"}}}}, Status: buildapiv1.BuildStatus{OutputDockerImageReference: "test/test-result:latest"}}
}
func TestMain(m *testing.M) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	DefaultPushOrPullRetryCount = 1
	DefaultPushOrPullRetryDelay = 5 * time.Millisecond
	flag.Parse()
	os.Exit(m.Run())
}
func TestDockerBuildError(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	expErr := errors.New("Artificial exception: Error building")
	s2iBuilder := newTestS2IBuilder(testS2IBuilderConfig{buildError: expErr})
	if err := s2iBuilder.Build(); err != expErr {
		t.Errorf("s2iBuilder.Build() = %v; want %v", err, expErr)
	}
}
func TestPushError(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	expErr := errors.New("Artificial exception: Error pushing image")
	s2iBuilder := newTestS2IBuilder(testS2IBuilderConfig{errPushImage: expErr})
	if err := s2iBuilder.Build(); !strings.HasSuffix(err.Error(), expErr.Error()) {
		t.Errorf("s2iBuilder.Build() = %v; want %v", err, expErr)
	}
}
func TestGetStrategyError(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	expErr := errors.New("Artificial exception: config error")
	s2iBuilder := newTestS2IBuilder(testS2IBuilderConfig{getStrategyErr: expErr})
	if err := s2iBuilder.Build(); err != expErr {
		t.Errorf("s2iBuilder.Build() = %v; want %v", err, expErr)
	}
}
func TestCopyToVolumeList(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	newArtifacts := []buildapiv1.ImageSourcePath{{SourcePath: "/path/to/source", DestinationDir: "path/to/destination"}}
	volumeList := s2iapi.VolumeList{s2iapi.VolumeSpec{Source: "/path/to/source", Destination: "path/to/destination"}}
	newVolumeList := copyToVolumeList(newArtifacts)
	if !reflect.DeepEqual(volumeList, newVolumeList) {
		t.Errorf("Expected artifacts mapping to match %#v, got %#v instead!", volumeList, newVolumeList)
	}
}
func TestInjectSecrets(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	secrets := []buildapiv1.SecretBuildSource{{Secret: corev1.LocalObjectReference{Name: "secret1"}, DestinationDir: "/tmp"}, {Secret: corev1.LocalObjectReference{Name: "secret2"}}}
	output := injectSecrets(secrets)
	for i, v := range output {
		secret := secrets[i]
		if v.Keep {
			t.Errorf("secret volume %s should not have been kept", secret.Secret.Name)
		}
		if secret.DestinationDir != v.Destination {
			t.Errorf("expected secret %s to be mounted to %s, got %s", secret.Secret.Name, secret.DestinationDir, v.Destination)
		}
	}
}
func TestInjectConfigMaps(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	configMaps := []buildapiv1.ConfigMapBuildSource{{ConfigMap: corev1.LocalObjectReference{Name: "configMap1"}, DestinationDir: "/tmp"}, {ConfigMap: corev1.LocalObjectReference{Name: "configMap2"}}}
	output := injectConfigMaps(configMaps)
	for i, v := range output {
		configMap := configMaps[i]
		if !v.Keep {
			t.Errorf("configMap volume %s should have been kept", configMap.ConfigMap.Name)
		}
		if configMap.DestinationDir != v.Destination {
			t.Errorf("expected configMap %s to be mounted to %s, got %s", configMap.ConfigMap.Name, configMap.DestinationDir, v.Destination)
		}
	}
}
func TestBuildEnvVars(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	expectedEnvList := s2iapi.EnvironmentList{s2iapi.EnvironmentSpec{Name: "OPENSHIFT_BUILD_NAME", Value: "openshift-test-1-build"}, s2iapi.EnvironmentSpec{Name: "OPENSHIFT_BUILD_NAMESPACE", Value: "openshift-demo"}, s2iapi.EnvironmentSpec{Name: "OPENSHIFT_BUILD_SOURCE", Value: "http://localhost/123"}, s2iapi.EnvironmentSpec{Name: "OPENSHIFT_BUILD_COMMIT", Value: "1575a90c569a7cc0eea84fbd3304d9df37c9f5ee"}, s2iapi.EnvironmentSpec{Name: "HTTPS_PROXY", Value: "https://test/secure:8443"}, s2iapi.EnvironmentSpec{Name: "HTTP_PROXY", Value: "http://test/insecure:8080"}}
	expectedLabelMap := map[string]string{"io.openshift.build.commit.id": "1575a90c569a7cc0eea84fbd3304d9df37c9f5ee", "io.openshift.build.name": "openshift-test-1-build", "io.openshift.build.namespace": "openshift-demo"}
	mockBuild := makeBuild()
	mockBuild.Name = "openshift-test-1-build"
	mockBuild.Namespace = "openshift-demo"
	mockBuild.Spec.Source.Git = &buildapiv1.GitBuildSource{URI: "http://localhost/123"}
	sourceInfo := &git.SourceInfo{}
	sourceInfo.CommitID = "1575a90c569a7cc0eea84fbd3304d9df37c9f5ee"
	resultedEnvList := buildEnvVars(mockBuild, sourceInfo)
	if !reflect.DeepEqual(expectedEnvList, resultedEnvList) {
		t.Errorf("Expected EnvironmentList to match:\n%#v\ngot:\n%#v", expectedEnvList, resultedEnvList)
	}
	resultedLabelList := buildLabels(mockBuild, sourceInfo)
	resultedLabelMap := map[string]string{}
	for _, label := range resultedLabelList {
		resultedLabelMap[label.Key] = label.Value
	}
	if !reflect.DeepEqual(expectedLabelMap, resultedLabelMap) {
		t.Errorf("Expected LabelList to match:\n%#v\ngot:\n%#v", expectedLabelMap, resultedLabelMap)
	}
}
func TestScriptProxyConfig(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	newBuild := &buildapiv1.Build{Spec: buildapiv1.BuildSpec{CommonSpec: buildapiv1.CommonSpec{Strategy: buildapiv1.BuildStrategy{SourceStrategy: &buildapiv1.SourceBuildStrategy{Env: append([]corev1.EnvVar{}, corev1.EnvVar{Name: "HTTPS_PROXY", Value: "https://test/secure"}, corev1.EnvVar{Name: "HTTP_PROXY", Value: "http://test/insecure"})}}}}}
	resultedProxyConf, err := scriptProxyConfig(newBuild)
	if err != nil {
		t.Fatalf("An error occurred while parsing the proxy config: %v", err)
	}
	if resultedProxyConf.HTTPProxy.Path != "/insecure" {
		t.Errorf("Expected HTTP Proxy path to be /insecure, got: %v", resultedProxyConf.HTTPProxy.Path)
	}
	if resultedProxyConf.HTTPSProxy.Path != "/secure" {
		t.Errorf("Expected HTTPS Proxy path to be /secure, got: %v", resultedProxyConf.HTTPSProxy.Path)
	}
}
func TestIncrementalPullError(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	inspectFunc := func(name string) (*docker.Image, error) {
		if name == "test/test-result:latest" {
			return nil, fmt.Errorf("no such image %s", name)
		}
		return &docker.Image{}, nil
	}
	pullFunc := func(opts docker.PullImageOptions, auth docker.AuthConfiguration) error {
		if strings.Contains(opts.Repository, "test/test-result") {
			return fmt.Errorf("image %s:%s does not exist", opts.Repository, opts.Tag)
		}
		return nil
	}
	s2ibuilder := newTestS2IBuilder(testS2IBuilderConfig{inspectImageFunc: inspectFunc, pullImageFunc: pullFunc})
	if err := s2ibuilder.Build(); err != nil {
		t.Errorf("unexpected build error: %v", err)
	}
}
func TestGetAssembleUser(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	testCases := []struct {
		name			string
		containerUser		string
		assembleUserLabel	string
		expectedResult		string
	}{{name: "empty"}, {name: "container user set", containerUser: "1002", expectedResult: "1002"}, {name: "assemble user label set", assembleUserLabel: "1003", expectedResult: "1003"}, {name: "assemble user override", containerUser: "1002", assembleUserLabel: "1003", expectedResult: "1003"}}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fakeDocker := &FakeDocker{inspectImageFunc: func(name string) (*docker.Image, error) {
				image := &docker.Image{ContainerConfig: docker.Config{User: tc.containerUser, Image: name, Labels: make(map[string]string)}}
				if len(tc.assembleUserLabel) > 0 {
					image.ContainerConfig.Labels[s2iconstants.AssembleUserLabel] = tc.assembleUserLabel
				}
				return image, nil
			}}
			assembleUser, err := getAssembleUser(fakeDocker, "dummy-image:latest")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if assembleUser != tc.expectedResult {
				t.Errorf("expected assemble user %s, got %s", tc.expectedResult, assembleUser)
			}
		})
	}
}
