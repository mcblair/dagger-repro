# dagger-repro

This repo contains my attempts to reproduce and document issues and behavior encountered with `dagger`.

- [dagger-repro](#dagger-repro)
  - [Weird Cache Invalidation Behavior](#weird-cache-invalidation-behavior)
    - [Background](#background)
    - [Observed behavior](#observed-behavior)
    - [Investigation](#investigation)
    - [Behavior Explained](#behavior-explained)
    - [Next Steps](#next-steps)

## Weird Cache Invalidation Behavior

### Background

On Monday 14 October 2024 I attempted to upgrade an application `go` module and the corresponding `dagger` module from `go1.23.1` to `go1.23.2`.

The [`.dagger`](.dagger/main.go) module is structured such that all dependency/toolchain versions can be maintained from a single place in code; specifically the root `dagger` module constructor like:

```golang
func New(
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
```

For the sake of user experience, a [`Makefile`](Makefile) is used to encapsulate common arguments, notably arguments that a required to be explicit, like host system files or directories outside of the `dagger` context. Like so:

```makefile
.PHONY: build
build: 
    @dagger call --ssh-dir ~/.ssh build binaries 
```

Lastly, because `git` behavior is such that the local filesystem timestamps do not match `commit` timestamps, a tool is used prior to `dagger` execution that ensures the local filesystem timestamps match the `commit` timestamps. Why? Initially this was implemented for the application so that `go test` caches aren't unnecessarily busted when *only* local filesystem timestamps changed. In a big project, this saves developers a significant amount of time when running unit tests. When `dagger` was implemented, it was decided that the same timestamp pattern can help avoid unnecessarily busting `dagger` caches of file/directory input arguments, like source dir.

### Observed behavior

When modifying the string `1.23.1` to `1.23.2` and calling `make build`, the following was observed:

1. The `dagger` module used cached results during `initialize` `installing module` in `.asModule` and `.initialize` function calls.
2. The `1.23.1` image of `go` was used during the user function call to `build binaries`.

Why wasn't the cache invalidated?

### Investigation

On Friday 18 October 2024, I reached out to Marco at `dagger` with regards to the observed behavior and after working through several isolated hello worlds, neither of us were able to reproduce the issue. The next task was to build this repro repo.

Starting with an empty repo, the `.dagger` module was constructed from scratch and original repo parity was added one piece at a time. Unable to reproduce the issue, I decided to go beyond `.dagger` parity and introduced the `Makefile` layer from the original repo; after adding `tools/git-mtimestamp`, I was finally able to reproduce the behavior!

### Behavior Explained

As explained in the background section, we sync local file system timestamps with `git` timestamps prior to every `Makefile` target execution. The *intention* was to only modify the timestamps for files that were unchanged, because we *do* want to invalidate caches when we modify or add new files. The *reality* was that the tool modified timestamps for changed/new files and the resulting behavior was a `dagger` cache hit. In other words, to `dagger` nothing changed, despite having modified `.dagger/main.go`.

To no surprise, when calling the `dagger` function directly, the behavior is entirely unreproducible: `dagger call --ssh-dir ~/.ssh build binaries`.

### Next Steps

In the immediate term, the `git-mtimestamp` tool needs:

- fixed such that it does not touch modified or new files
- tests

Looking ahead I going to re-evaluate our general approach to syncing local file system timestamps with `git` timestamps.
> **Note:** I will not be fixing the repro implementation in this repository as I'd prefer to preserve the reproducible behavior.
