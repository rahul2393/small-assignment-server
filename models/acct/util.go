package acct

import (
	"github.com/pkg/errors"
)

var (
	// ErrMergeWrongType occurs when the merge function is
	// called with an incompatible type.
	ErrMergeWrongType = errors.New("model: merge attempted with incompatible types")
)
