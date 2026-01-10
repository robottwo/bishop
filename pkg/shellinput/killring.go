package shellinput

const (
	killRingMax = 30
)

// killDirection represents the direction of a kill operation
type killDirection int

const (
	killDirectionUnknown killDirection = iota
	killDirectionForward
	killDirectionBackward
)

// KillRing manages the Emacs-style kill ring for cut/paste operations.
// It stores recently killed (cut) text and supports yank (paste) and yank-pop
// (cycling through previous kills) operations.
type KillRing struct {
	// ring stores recently killed text. The head (index 0) is the most recent kill.
	ring [][]rune
	// index is used when cycling through the ring with yank-pop
	index int
	// lastDirection tracks the direction of the previous kill to support
	// Bash/zsh-style kill ring appending semantics
	lastDirection killDirection
	// lastWasKill tracks whether the last command was a kill operation
	lastWasKill bool
	// yankActive indicates whether a yank operation is currently active
	yankActive bool
	// yankStart is the start position of the last yank in the input buffer
	yankStart int
	// yankEnd is the end position of the last yank in the input buffer
	yankEnd int
}
