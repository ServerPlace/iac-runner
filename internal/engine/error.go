package engine

import "fmt"

type ErrBlocked struct {
	Step   string
	Reason string
}

func (e ErrBlocked) Error() string {
	return fmt.Sprintf("%s blocked: %s", e.Step, e.Reason)
}
