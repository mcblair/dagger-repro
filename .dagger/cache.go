package main

const (
	BinariesCache     = "binaries"
	GoModCache        = "go-mod"
	GoBuildCache      = "go-build"
	PulumiPluginCache = "pulumi-plugin-cache"

	// Used by dagger itself, not directly in our dagger module
	ModGoBuildCache = "modgobuildcache"
	ModGoModCache   = "modgomodcache"
)

var caches = map[string]string{
	BinariesCache:     "/work/src/bin",
	GoModCache:        "/go/pkg/mod",
	GoBuildCache:      "/go/build-cache",
	PulumiPluginCache: "/root/.pulumi/plugins",

	// Used by dagger itself, not directly in our dagger module
	ModGoBuildCache: "/go/build-cache",
	ModGoModCache:   "/go/pkg/mod",
}

type Cache struct {
	// +Private
	DaggerRepro *DaggerRepro
}

func (m *DaggerRepro) Cache() *Cache {
	return &Cache{
		DaggerRepro: m,
	}
}
