module github.com/abhinav/tmux-fastcopy/integration

go 1.20

require (
	github.com/abhinav/tmux-fastcopy v0.10.0
	github.com/creack/pty v1.1.18
	github.com/jaguilar/vt100 v0.0.0-20201024211400-81de19cb81a4
	github.com/stretchr/testify v1.8.2
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/abhinav/tmux-fastcopy => ../
