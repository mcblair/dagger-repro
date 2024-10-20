package main

import (
	"github.com/mcblair/dagger-repro/.dagger/internal/dagger"
)

// Build commands.
func (m *DaggerRepro) Build() *Build {
	return &Build{
		DaggerRepro: m,
	}
}

type Build struct {
	// +private
	DaggerRepro *DaggerRepro
}

func (m *Build) Binaries() *dagger.Container {
	// Build cli
	cli := m.DaggerRepro.Build().Tool("go-builder")

	return m.DaggerRepro.GoToolchain.BuildEnv(dag.Directory().
		WithDirectory("/", m.DaggerRepro.Source, dagger.DirectoryWithDirectoryOpts{
			Exclude: []string{
				".dagger",
				".git",
			},
		})).
		WithMountedCache(caches[BinariesCache], dag.CacheVolume(BinariesCache), dagger.ContainerWithMountedCacheOpts{
			Sharing: dagger.Shared,
		}).
		WithMountedFile("/usr/local/bin/cli", cli).
		WithExec([]string{"cli", "build", ".", caches[BinariesCache]})
}

func (m *Build) Tool(toolName string) *dagger.File {
	return m.DaggerRepro.GoToolchain.BuildEnv(dag.Directory().
		WithDirectory("/", m.DaggerRepro.Source, dagger.DirectoryWithDirectoryOpts{
			Include: []string{
				"tools/" + toolName + "/**",
				"go.mod",
				"go.sum",
			},
		})).
		WithExec([]string{"go", "build", "-buildvcs=false", "-trimpath", "-o", "bin/" + toolName, "./tools/" + toolName}).
		File("bin/" + toolName)
}
