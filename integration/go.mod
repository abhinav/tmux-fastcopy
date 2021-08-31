module github.com/abhinav/tmux-fastcopy/integration

go 1.17

require (
	github.com/abhinav/tmux-fastcopy v0.0.0-00010101000000-000000000000
	github.com/creack/pty v1.1.15
	github.com/jaguilar/vt100 v0.0.0-20201024211400-81de19cb81a4
	github.com/stretchr/testify v1.7.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace github.com/abhinav/tmux-fastcopy => ../
