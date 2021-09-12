package tmux

//go:generate mockgen -destination tmuxtest/mock_driver.go -package tmuxtest github.com/abhinav/tmux-fastcopy/internal/tmux Driver
