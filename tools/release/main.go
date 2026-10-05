package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// ===== Types =====

type Target struct {
	Name         string
	Availability func() string
	Package      func(release *Release, archivePath string) error
	Extension    string
}

type Release struct {
	Version   string
	WorkDir   string
	SourceDir string
}

// ===== Constants =====

const (
	distDirectory    = "dist"
	appName          = "Hordefall"
	binaryName       = "hordefall"
	allTargets       = "all"
	stripFlags       = "-s -w"
	windowsGUIFlag   = " -H windowsgui"
	dockerGoImage    = "golang:1.26"
	linuxPackages    = "libgl1-mesa-dev xorg-dev libasound2-dev"
	availableMessage = ""
)

var targets = []Target{
	{Name: "windows-amd64", Availability: alwaysAvailable, Package: packageWindows("amd64"), Extension: ".zip"},
	{Name: "windows-arm64", Availability: alwaysAvailable, Package: packageWindows("arm64"), Extension: ".zip"},
	{Name: "macos-universal", Availability: alwaysAvailable, Package: packageMacOS, Extension: ".tar.gz"},
	{Name: "linux-amd64", Availability: linuxAvailability, Package: packageLinux, Extension: ".tar.gz"},
	{Name: "freebsd-amd64", Availability: nativeAvailability("freebsd"), Package: packageNative("freebsd"), Extension: ".tar.gz"},
}

// ===== Public API =====

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "release:", err)
		os.Exit(1)
	}
}

// ===== Internal =====

func run() error {
	requested := flag.String("targets", allTargets, "comma-separated targets: "+targetNames())
	flag.Parse()
	sourceDir, err := os.Getwd()
	if err != nil {
		return err
	}
	workDir, err := os.MkdirTemp("", "hordefall-release-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workDir)
	if err := os.MkdirAll(distDirectory, 0o755); err != nil {
		return err
	}
	release := &Release{Version: gitVersion(), WorkDir: workDir, SourceDir: sourceDir}
	failures := 0
	for _, target := range selectTargets(*requested) {
		failures += buildTarget(release, target)
	}
	if failures > 0 {
		return fmt.Errorf("%d target(s) failed", failures)
	}
	return nil
}

func buildTarget(release *Release, target Target) int {
	reason := target.Availability()
	if reason != availableMessage {
		fmt.Printf("skip   %-16s %s\n", target.Name, reason)
		return 0
	}
	archivePath := filepath.Join(distDirectory, fmt.Sprintf("%s-%s-%s%s", appName, release.Version, target.Name, target.Extension))
	fmt.Printf("build  %-16s ...\n", target.Name)
	if err := target.Package(release, archivePath); err != nil {
		fmt.Printf("FAILED %-16s %v\n", target.Name, err)
		return 1
	}
	fmt.Printf("done   %-16s %s\n", target.Name, archivePath)
	return 0
}

func selectTargets(requested string) []Target {
	if requested == allTargets {
		return targets
	}
	names := strings.Split(requested, ",")
	selected := make([]Target, 0, len(names))
	for _, target := range targets {
		selected = appendIf(selected, target, slices.Contains(names, target.Name))
	}
	return selected
}

func targetNames() string {
	names := make([]string, 0, len(targets))
	for _, target := range targets {
		names = append(names, target.Name)
	}
	return strings.Join(names, ", ")
}

func gitVersion() string {
	output, err := exec.Command("git", "describe", "--tags", "--always", "--dirty").Output()
	if err != nil {
		return "dev"
	}
	return strings.TrimSpace(string(output))
}

func alwaysAvailable() string {
	return availableMessage
}

func nativeAvailability(goos string) func() string {
	return func() string {
		if runtime.GOOS == goos {
			return availableMessage
		}
		return "needs cgo and the system graphics libraries: run this tool on " + goos
	}
}

func linuxAvailability() string {
	if runtime.GOOS == "linux" {
		return availableMessage
	}
	if exec.Command("docker", "info").Run() == nil {
		return availableMessage
	}
	return "needs cgo and X11/OpenGL: run this tool on Linux, or start Docker"
}

func compile(release *Release, goos, goarch string, isCgo bool, ldflags, output string) error {
	command := exec.Command("go", "build", "-trimpath", "-ldflags", ldflags, "-o", output, ".")
	command.Dir = release.SourceDir
	command.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch, "CGO_ENABLED="+map[bool]string{false: "0", true: "1"}[isCgo])
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	return command.Run()
}

func packageWindows(goarch string) func(release *Release, archivePath string) error {
	return func(release *Release, archivePath string) error {
		binary := filepath.Join(release.WorkDir, "windows-"+goarch+".exe")
		if err := compile(release, "windows", goarch, false, stripFlags+windowsGUIFlag, binary); err != nil {
			return err
		}
		return writeZip(archivePath, []ArchiveEntry{{Name: appName + "/" + appName + ".exe", Mode: 0o755, SourcePath: binary}})
	}
}

func packageNative(goos string) func(release *Release, archivePath string) error {
	return func(release *Release, archivePath string) error {
		binary := filepath.Join(release.WorkDir, goos+"-"+runtime.GOARCH)
		if err := compile(release, goos, runtime.GOARCH, true, stripFlags, binary); err != nil {
			return err
		}
		return writeTarGz(archivePath, []ArchiveEntry{{Name: appName + "/" + binaryName, Mode: 0o755, SourcePath: binary}})
	}
}

func packageLinux(release *Release, archivePath string) error {
	if runtime.GOOS == "linux" {
		return packageNative("linux")(release, archivePath)
	}
	binary := filepath.Join(release.WorkDir, "linux-amd64")
	if err := compileLinuxInDocker(release, binary); err != nil {
		return err
	}
	return writeTarGz(archivePath, []ArchiveEntry{{Name: appName + "/" + binaryName, Mode: 0o755, SourcePath: binary}})
}

func compileLinuxInDocker(release *Release, output string) error {
	script := fmt.Sprintf("apt-get update -qq && apt-get install -y -qq %s >/dev/null && go build -buildvcs=false -trimpath -ldflags '%s' -o /out/%s .", linuxPackages, stripFlags, filepath.Base(output))
	command := exec.Command("docker", "run", "--rm", "--platform", "linux/amd64",
		"-v", filepath.ToSlash(release.SourceDir)+":/src", "-v", filepath.ToSlash(filepath.Dir(output))+":/out",
		"-w", "/src", dockerGoImage, "bash", "-c", script)
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	return command.Run()
}

func appendIf[T any](items []T, item T, shouldAppend bool) []T {
	if !shouldAppend {
		return items
	}
	return append(items, item)
}
