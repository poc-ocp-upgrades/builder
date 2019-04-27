package dockerfile

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"github.com/docker/docker/builder/dockerfile/command"
)

func TestWrite(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	testCases := map[string]struct {
		in	string
		want	string
	}{"empty input": {in: ``, want: ``}, "only comments": {in: `# This is a comment
# and this is another comment
	# while this is an indented comment`, want: ``}, "simple Dockerfile": {in: `FROM scratch
LABEL version=1.0
FROM busybox
ENV PATH=/bin
`, want: `FROM scratch
LABEL version=1.0
FROM busybox
ENV PATH=/bin
`}, "Dockerfile with comments": {in: `# This is a Dockerfile
FROM scratch
LABEL version=1.0
# Here we start building a second image
FROM busybox
ENV PATH=/bin
`, want: `FROM scratch
LABEL version=1.0
FROM busybox
ENV PATH=/bin
`}, "all Dockerfile instructions": {in: `FROM busybox:latest
MAINTAINER nobody@example.com
ONBUILD ADD . /app/src
ONBUILD RUN echo "Hello universe!"
LABEL version=1.0
EXPOSE 8080
VOLUME /var/run/www
ENV PATH=/bin TEST=
ADD file /home/
COPY dir/ /tmp/
FROM other as 2
COPY --from=test /a /b
RUN echo "Hello world!"
ENTRYPOINT /bin/sh
CMD ["-c", "env"]
USER 1001
WORKDIR /home
`, want: `FROM busybox:latest
MAINTAINER nobody@example.com
ONBUILD ADD . /app/src
ONBUILD RUN echo "Hello universe!"
LABEL version=1.0
EXPOSE 8080
VOLUME /var/run/www
ENV PATH=/bin TEST=
ADD file /home/
COPY dir/ /tmp/
FROM other as 2
COPY --from=test /a /b
RUN echo "Hello world!"
ENTRYPOINT /bin/sh
CMD ["-c","env"]
USER 1001
WORKDIR /home
`}}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			node, err := Parse(strings.NewReader(tc.in))
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			got := Write(node)
			want := []byte(tc.want)
			if !bytes.Equal(got, want) {
				t.Errorf("got:\n%swant:\n%s", got, want)
			}
		})
	}
}
func TestParseTreeToDockerfileNilNode(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	got := Write(nil)
	if got != nil {
		t.Errorf("Write(nil) = %#v; want nil", got)
	}
}
func TestFindAll(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	instructions := `FROM scratch
LABEL version=1.0
FROM busybox
ENV PATH=/bin
`
	node, err := Parse(strings.NewReader(instructions))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	for cmd, want := range map[string][]int{command.From: {0, 2}, command.Label: {1}, command.Env: {3}, command.Maintainer: nil, "UnknownCommand": nil} {
		got := FindAll(node, cmd)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("FindAll(node, %q) = %#v; want %#v", cmd, got, want)
		}
	}
}
func TestFindAllNilNode(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	cmd := command.From
	got := FindAll(nil, cmd)
	if got != nil {
		t.Errorf("FindAll(nil, %q) = %#v; want nil", cmd, got)
	}
}
func TestInsertInstructions(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	testCases := map[string]struct {
		original	string
		index		int
		newInstructions	string
		want		string
	}{"insert nothing": {original: `FROM busybox
ENV PATH=/bin
`, index: 0, newInstructions: ``, want: `FROM busybox
ENV PATH=/bin
`}, "insert instruction in empty file": {original: ``, index: 0, newInstructions: `FROM busybox`, want: `FROM busybox
`}, "prepend single instruction": {original: `FROM busybox
ENV PATH=/bin
`, index: 0, newInstructions: `FROM scratch`, want: `FROM scratch
FROM busybox
ENV PATH=/bin
`}, "append single instruction": {original: `FROM busybox
ENV PATH=/bin
`, index: 2, newInstructions: `FROM scratch`, want: `FROM busybox
ENV PATH=/bin
FROM scratch
`}, "insert single instruction in the middle": {original: `FROM busybox
ENV PATH=/bin
`, index: 1, newInstructions: `LABEL version=1.0`, want: `FROM busybox
LABEL version=1.0
ENV PATH=/bin
`}}
	for name, tc := range testCases {
		got, err := Parse(strings.NewReader(tc.original))
		if err != nil {
			t.Errorf("InsertInstructions: %s: parse error: %v", name, err)
			continue
		}
		err = InsertInstructions(got, tc.index, tc.newInstructions)
		if err != nil {
			t.Errorf("InsertInstructions: %s: %v", name, err)
			continue
		}
		want, err := Parse(strings.NewReader(tc.want))
		if err != nil {
			t.Errorf("InsertInstructions: %s: parse error: %v", name, err)
			continue
		}
		if !bytes.Equal(Write(got), Write(want)) {
			t.Errorf("InsertInstructions: %s: got %#v; want %#v", name, got, want)
		}
	}
}
func TestInsertInstructionsNilNode(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	err := InsertInstructions(nil, 0, "")
	if err == nil {
		t.Errorf("InsertInstructions: got nil; want error")
	}
}
func TestInsertInstructionsPosOutOfRange(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	original := `FROM busybox
ENV PATH=/bin
`
	node, err := Parse(strings.NewReader(original))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	for _, pos := range []int{-1, 3, 4} {
		err := InsertInstructions(node, pos, "")
		if err == nil {
			t.Errorf("InsertInstructions(node, %d, \"\"): got nil; want error", pos)
		}
	}
}
func TestInsertInstructionsUnparseable(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	original := `FROM busybox
ENV PATH=/bin
`
	node, err := Parse(strings.NewReader(original))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	for name, instructions := range map[string]string{"env without value": `ENV PATH`, "nested json": `CMD [ "echo", [ "nested json" ] ]`} {
		err = InsertInstructions(node, 1, instructions)
		if err == nil {
			t.Errorf("InsertInstructions: %s: got nil; want error", name)
		}
	}
}
func TestBaseImages(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	testCases := map[string]struct {
		in	string
		want	[]string
	}{"empty Dockerfile": {in: ``, want: nil}, "FROM missing argument": {in: `FROM`, want: nil}, "single FROM": {in: `FROM centos:7`, want: []string{"centos:7"}}, "multiple FROM": {in: `FROM scratch
COPY . /boot
FROM centos:7`, want: []string{"scratch", "centos:7"}}}
	for name, tc := range testCases {
		node, err := Parse(strings.NewReader(tc.in))
		if err != nil {
			t.Errorf("%s: parse error: %v", name, err)
			continue
		}
		got := baseImages(node)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("baseImages: %s: got %#v; want %#v", name, got, tc.want)
		}
	}
}
func TestBaseImagesNilNode(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	if got := baseImages(nil); got != nil {
		t.Errorf("baseImages(nil) = %#v; want nil", got)
	}
}
func TestExposedPorts(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	testCases := map[string]struct {
		in	string
		want	[][]string
	}{"empty Dockerfile": {in: ``, want: nil}, "EXPOSE missing argument": {in: `EXPOSE`, want: nil}, "EXPOSE no FROM": {in: `EXPOSE 8080`, want: nil}, "single EXPOSE after FROM": {in: `FROM centos:7
		EXPOSE 8080`, want: [][]string{{"8080"}}}, "multiple EXPOSE and FROM": {in: `# EXPOSE before FROM should be ignore
EXPOSE 777
FROM busybox
EXPOSE 8080
COPY . /boot
FROM rhel
# no EXPOSE instruction
FROM centos:7
EXPOSE 8000
EXPOSE 9090 9091
`, want: [][]string{{"8080"}, nil, {"8000", "9090", "9091"}}}}
	for name, tc := range testCases {
		node, err := Parse(strings.NewReader(tc.in))
		if err != nil {
			t.Errorf("%s: parse error: %v", name, err)
			continue
		}
		got := exposedPorts(node)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("exposedPorts: %s: got %#v; want %#v", name, got, tc.want)
		}
	}
}
func TestExposedPortsNilNode(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	if got := exposedPorts(nil); got != nil {
		t.Errorf("exposedPorts(nil) = %#v; want nil", got)
	}
}
func TestNextValues(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	testCases := map[string][]string{`FROM busybox:latest`: {"busybox:latest"}, `MAINTAINER nobody@example.com`: {"nobody@example.com"}, `LABEL version=1.0`: {"version", "1.0"}, `EXPOSE 8080`: {"8080"}, `VOLUME /var/run/www`: {"/var/run/www"}, `ENV PATH=/bin`: {"PATH", "/bin"}, `ADD file /home/`: {"file", "/home/"}, `COPY dir/ /tmp/`: {"dir/", "/tmp/"}, `RUN echo "Hello world!"`: {`echo "Hello world!"`}, `ENTRYPOINT /bin/sh`: {"/bin/sh"}, `CMD ["-c", "env"]`: {"-c", "env"}, `USER 1001`: {"1001"}, `WORKDIR /home`: {"/home"}}
	for original, want := range testCases {
		node, err := Parse(strings.NewReader(original))
		if err != nil {
			t.Fatalf("parse error: %s: %v", original, err)
		}
		if len(node.Children) != 1 {
			t.Fatalf("unexpected number of children in test case: %s", original)
		}
		node = node.Children[0]
		if got := nextValues(node); !reflect.DeepEqual(got, want) {
			t.Errorf("nextValues(%+v) = %#v; want %#v", node, got, want)
		}
	}
}
func TestNextValuesOnbuild(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	testCases := map[string][]string{`ONBUILD ADD . /app/src`: {".", "/app/src"}, `ONBUILD RUN echo "Hello universe!"`: {`echo "Hello universe!"`}}
	for original, want := range testCases {
		node, err := Parse(strings.NewReader(original))
		if err != nil {
			t.Fatalf("parse error: %s: %v", original, err)
		}
		if len(node.Children) != 1 {
			t.Fatalf("unexpected number of children in test case: %s", original)
		}
		node = node.Children[0].Next
		if node == nil || len(node.Children) != 1 {
			t.Fatalf("unexpected number of children in ONBUILD instruction of test case: %s", original)
		}
		node = node.Children[0]
		if got := nextValues(node); !reflect.DeepEqual(got, want) {
			t.Errorf("nextValues(%+v) = %#v; want %#v", node, got, want)
		}
	}
}
func TestNextValuesNilNode(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	if got := nextValues(nil); got != nil {
		t.Errorf("nextValues(nil) = %#v; want nil", got)
	}
}
