module github.com/redneckbeard/thanos

go 1.18

require (
	github.com/fatih/color v1.13.0
	github.com/redneckbeard/thanos/stdlib v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.3.0
	golang.org/x/tools v0.1.9
)

replace github.com/redneckbeard/thanos/stdlib => ./stdlib

require (
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/crypto v0.0.0-20211215153901-e495a2d5b3d3 // indirect
	golang.org/x/mod v0.5.1 // indirect
	golang.org/x/sys v0.0.0-20211205182925-97ca703d548d // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
)
