package controller

import (
	"fmt"
	"github.com/ServerPlace/iac-controller/pkg/api"
	"github.com/rs/zerolog"
)

const (
	ModePlan  = "plan"
	ModeApply = "apply"
)

func logFormat(request *api.CredentialsRequest, level zerolog.Level, msg string) string {
	var logMessage string
	switch level {
	default:
		logMessage = fmt.Sprintf(
			`Repository: %s
Mode: %s
CheckedOutSHA: %s
PR Number: %s
Msg: %s
`,
			request.Repo,
			request.Mode,
			request.HeadSHA,
			request.PRNumber,
			msg)
	}
	return logMessage
}
