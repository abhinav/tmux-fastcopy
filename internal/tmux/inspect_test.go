package tmux_test

import (
	"testing"

	"github.com/abhinav/tmux-fastcopy/internal/tmux"
	"github.com/abhinav/tmux-fastcopy/internal/tmux/tmuxtest"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInspectPane(t *testing.T) {
	t.Parallel()

	message := []byte("%42\t@123\t80\t40\tcopy-mode\t40\t0\t/home/user/dir")

	ctrl := gomock.NewController(t)
	mockTmux := tmuxtest.NewMockDriver(ctrl)

	mockTmux.EXPECT().
		DisplayMessage(gomock.Any()).
		Return(message, nil)

	got, err := tmux.InspectPane(mockTmux, "foo")
	require.NoError(t, err)
	assert.Equal(t, &tmux.PaneInfo{
		ID:             "%42",
		WindowID:       "@123",
		Width:          80,
		Height:         40,
		Mode:           tmux.CopyMode,
		ScrollPosition: 40,
		CurrentPath:    "/home/user/dir",
	}, got)

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		s := got.String()
		assert.Contains(t, s, "id: %42")
		assert.Contains(t, s, "windowID: @123")
		assert.Contains(t, s, "width: 80")
		assert.Contains(t, s, "height: 40")
		assert.Contains(t, s, "mode: copy-mode")
		assert.Contains(t, s, "scrollPosition: 40")
		assert.Contains(t, s, "currentPath: /home/user/dir")
	})
}
