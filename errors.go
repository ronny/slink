package slink

type ErrInvalidLinkURL struct {
	msg string
}

func (e *ErrInvalidLinkURL) Error() string {
	return e.msg
}

type ErrInvalidShortLinkID struct {
	msg string
}

func (e *ErrInvalidShortLinkID) Error() string {
	return e.msg
}
