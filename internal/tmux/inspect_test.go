package tmux_test

import (
	"testing"

	"github.com/abhinav/tmux-fastcopy/internal/tmux"
	"github.com/abhinav/tmux-fastcopy/internal/tmux/tmuxtest"
	"github.com/golang/mock/gomock"
	"github.com/maxatome/go-testdeep/td"
)

func TestInspectPane(t *testing.T) {
	t.Parallel()

	message := []byte("%42\t@123\t80\t40\tcopy-mode\t40")

	ctrl := gomock.NewController(t)
	mockTmux := tmuxtest.NewMockDriver(ctrl)

	mockTmux.EXPECT().
		DisplayMessage(gomock.Any()).
		Return(message, nil)

	got, err := tmux.InspectPane(mockTmux, "foo")
	td.CmpNoError(t, err)
	td.Cmp(t, got, &tmux.PaneInfo{
		ID:             "%42",
		WindowID:       "@123",
		Width:          80,
		Height:         40,
		Mode:           tmux.CopyMode,
		ScrollPosition: 40,
	})
}