package tmuxtest

import (
	"fmt"

	"github.com/abhinav/tmux-fastcopy/internal/tmux"
	"github.com/golang/mock/gomock"
)

// DisplayMessageRequestMatcher is a gomock matcher that matches
// tmux.DisplayMessageRequest objects by pane ID.
type DisplayMessageRequestMatcher struct {
	Pane string
}

var _ gomock.Matcher = DisplayMessageRequestMatcher{}

func (m DisplayMessageRequestMatcher) String() string {
	return fmt.Sprintf("DisplayMessageRequest{Pane: %q}", m.Pane)
}

// Matches reports whether the provided DisplayMessageRequest matches.
func (m DisplayMessageRequestMatcher) Matches(x interface{}) bool {
	req, ok := x.(tmux.DisplayMessageRequest)
	if !ok {
		return false
	}

	return req.Pane == m.Pane
}
