package types

import "regexp"

// sourceFrame matches the " [<path>.go:<line>]" annotation that
// cosmossdk.io/errors appends when one of its errors is formatted with %v or
// %w (fmt.Errorf("...: %w", err) bakes it into the new error's text). The path
// is the build host's (module cache or -trimpath form), so text carrying it
// differs between two nodes built differently.
var sourceFrame = regexp.MustCompile(` \[[^\[\]]*\.go:[0-9]+\]`)

// PlainErrorText is err's text without source-file annotations, for every
// place where error text becomes consensus data: revert data handed to an EVM
// caller, the error recorded on a failed queued call. It is a function of the
// text alone, so every node computes the same result whatever its build.
func PlainErrorText(err error) string {
	if err == nil {
		return ""
	}
	return sourceFrame.ReplaceAllString(err.Error(), "")
}

// PlainError is an error whose text is fixed when it is built (without
// source-file annotations) and which still unwraps to the error it reports,
// so errors.Is keeps working (an EVM precompile relies on that to tell out of
// gas from a revert).
type PlainError struct {
	Text  string
	Cause error
}

func (e PlainError) Error() string { return e.Text }
func (e PlainError) Unwrap() error { return e.Cause }

// NewPlainError is prefix + ": " + PlainErrorText(cause), unwrapping to cause.
func NewPlainError(prefix string, cause error) error {
	text := PlainErrorText(cause)
	if prefix != "" {
		text = prefix + ": " + text
	}
	return PlainError{Text: text, Cause: cause}
}
