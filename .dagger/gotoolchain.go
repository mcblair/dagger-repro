package main

import (
	"github.com/mcblair/dagger-repro/.dagger/internal/dagger"
)

type GoToolchain struct {
	// Go version
	Version string
	// PAT for github access.
	GithubToken *dagger.Secret
	// Directory containing SSH credentials for github access. (e.g. ~/.ssh)
	SshDir *dagger.Directory
}

func NewGoToolchain(
	// Go version to use.
	version string,
	githubToken *dagger.Secret,
	sshDir *dagger.Directory,
) *GoToolchain {
	return &GoToolchain{
		Version:     version,
		GithubToken: githubToken,
		SshDir:      sshDir,
	}
}

func (g *GoToolchain) Base() *dagger.Container {
	return dag.Container(dagger.ContainerOpts{
		Platform: "linux/arm64",
	}).
		From("golang:"+g.Version).
		WithEnvVariable("GOOS", "linux").
		WithEnvVariable("GOARCH", "arm64").
		WithEnvVariable("CGO_ENABLED", "0").
		WithEnvVariable("GO111MODULE", "on").
		WithEnvVariable("GOPRIVATE", "github.com/mcblair").
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "-y", "npm"}).
		WithExec([]string{"go", "install", "gotest.tools/gotestsum@latest"}).
		WithExec([]string{"curl", "-sSfL", "https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh", "-o", "golangci-lint-install.sh"}).
		WithExec([]string{"sh", "golangci-lint-install.sh", "-b", "/bin"})
}

func (g *GoToolchain) EnvBase() *dagger.Container {
	return g.Base().
		WithExec([]string{"sh", "-c", "echo \"deb http://security.debian.org/debian-security buster/updates main\" >> /etc/apt/sources.list"}).
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "-y", "fontconfig", "xfonts-75dpi", "xfonts-base", "libssl1.1"}).
		WithExec([]string{"curl", "-sSfL", "https://github.com/wkhtmltopdf/packaging/releases/download/0.12.6.1-2/wkhtmltox_0.12.6.1-2.bullseye_arm64.deb", "-o", "wkhtmltox_0.12.6.1-2.bullseye_arm64.deb"}).
		WithExec([]string{"dpkg", "-i", "wkhtmltox_0.12.6.1-2.bullseye_arm64.deb"})
}

func (g *GoToolchain) Env(source *dagger.Directory) *dagger.Container {
	base := g.EnvBase().
		WithEnvVariable("GOMODCACHE", caches[GoModCache]).
		WithMountedCache(
			caches[GoModCache],
			dag.CacheVolume(GoModCache),
			dagger.ContainerWithMountedCacheOpts{
				Sharing: dagger.Shared,
			}).
		WithWorkdir("/work/src").
		// run `go mod download` with only go.mod files (re-run only if mod files have changed)
		WithDirectory("/work/src", source, dagger.ContainerWithDirectoryOpts{
			Include: []string{"go.mod", "go.sum"},
		})

	if g.GithubToken != nil {
		base = base.
			WithSecretVariable("GITHUB_TOKEN", g.GithubToken).
			WithExec([]string{"sh", "-c", "git config --global --add url.https://$GITHUB_TOKEN@github.com/.insteadOf https://github.com/"})
	} else if g.SshDir != nil {
		base = base.
			WithMountedDirectory("/root/.ssh", g.SshDir).
			WithExec([]string{"git", "config", "--global", "--add", "url.git@github.com:.insteadOf", "https://github.com/"})
	} else {
		return base.WithExec([]string{"echo", "either ssh or pat is required", "exit", "1"})
	}

	return base.
		WithExec([]string{"go", "mod", "download"}).
		WithMountedDirectory("/work/src", source)
}

func (g *GoToolchain) BuildEnv(source *dagger.Directory) *dagger.Container {
	return g.Env(source).
		WithEnvVariable("GOCACHE", caches[GoBuildCache]).
		WithMountedCache(
			caches[GoBuildCache],
			dag.CacheVolume(GoBuildCache),
			dagger.ContainerWithMountedCacheOpts{
				Sharing: dagger.Shared,
			})
}
