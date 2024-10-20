package main

import (
	"context"
	"errors"

	"github.com/mcblair/dagger-repro/.dagger/internal/dagger"
)

type DaggerRepro struct {
	// +private
	Source *dagger.Directory
	// +private
	GoToolchain *GoToolchain
}

func New(
	ctx context.Context,
	// Directory containing source code.
	// +optional
	// +defaultPath="/"
	// +ignore=[".dagger", ".github"]
	source *dagger.Directory,
	// Secret containing Github PAT for cloning private GH repos on CI machines.
	// +optional
	githubToken *dagger.Secret,
	// Directory containing SSH credentials for cloning private GH repos on dev machines.
	// +optional
	sshDir *dagger.Directory,
) (*DaggerRepro, error) {
	if githubToken == nil && sshDir == nil {
		return nil, errors.New("gh-token or ssh-dir is required")
	} else if githubToken != nil && sshDir != nil {
		return nil, errors.New("gh-token and ssh-dir cannot be used together")
	}

	return &DaggerRepro{
		Source:      source,
		GoToolchain: NewGoToolchain("1.23.1", githubToken, sshDir),
	}, nil
}
