module github.com/barisgit/goflux/cli

go 1.24.2

require (
	github.com/AlecAivazis/survey/v2 v2.3.7
	github.com/barisgit/goflux v0.1.10
	github.com/creack/pty v1.1.24
	github.com/fsnotify/fsnotify v1.9.0
	github.com/mattn/go-isatty v0.0.20
	github.com/spf13/cobra v1.9.1
	gopkg.in/yaml.v3 v3.0.1
)

// Local development - replace with local framework
replace github.com/barisgit/goflux => ../

require (
	github.com/danielgtaylor/huma/v2 v2.32.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/term v0.32.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)
