module github.com/abhinav/tmux-fastcopy/integration

go 1.24

require (
	github.com/abhinav/tmux-fastcopy v0.14.1
	github.com/creack/pty v1.1.24
	github.com/stretchr/testify v1.10.0
	go.abhg.dev/io/ioutil v0.1.0
	go.uber.org/multierr v1.11.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/abhinav/tmux-fastcopy => ../
