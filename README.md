# gocovercache
coverage report cache for golang

# Why?
go1.5 build takes longer time than go1.4.
see https://golang.org/doc/go1.5#performance

# Installation
```
go get github.com/azihsoyn/gocovercache
```

# Ussge
```
Usage of gocovercache:
  -coverprofile string
        coverage report file (default "profile.cov")
  -outdir string
        cache directory (default ".cache")
  -parallel int
        parallel number (default cpus)
  -v    verbose output
```

# For CircleCI
```yaml:
machine:
  pre:
  - mkdir -p /home/ubuntu/cache

dependencies:
  cache_directories:
    - /home/ubuntu/cache
  post:
    # to cache report you should test at dependencies:post: phase
    # see https://discuss.circleci.com/t/overriding-go-inference-in-the-dependencies-phase/660
    - go get github.com/mattn/goveralls
    - go get golang.org/x/tools/cmd/cover
    - go get github.com/azihsoyn/gocovercache
    - gocovercache -outdir=/home/ubuntu/cache -coverprofile=profile.cov
    - goveralls -coverprofile=profile.cov -service=circle-ci -repotoken $COVERALLS_TOKEN

test:
  override:
    - go vet ./...
    # if you need detect race condition
    - go test -race ./...
```

# ToDO
- test! test! test!
- refactor
- support specific pkgs

# Not ToDo
- support -race option # too cost
