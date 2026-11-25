/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package validation

import (
	"fmt"
	"strings"
)

type ghAnnotationLevel string

const (
	ghAnnotationNotice  ghAnnotationLevel = "notice"
	ghAnnotationWarning ghAnnotationLevel = "warning"
	ghAnnotationError   ghAnnotationLevel = "error"
)

func writeGhAnnotation(level ghAnnotationLevel, file string, lines ...string) {
	fmt.Printf("::%s file=%s::%s\n", level, file, strings.Join(lines, "%0A"))
}
