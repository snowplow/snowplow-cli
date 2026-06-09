/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package events

import (
	snplog "github.com/snowplow-product/snowplow-cli/internal/logging"
	"github.com/spf13/cobra"
)

var EventsCmd = &cobra.Command{
	Use:     "events",
	Short:   "Work with Snowplow events",
	Example: `  $ snowplow-cli events send --collector collector.example.com --sdjson '{"schema":"iglu:com.snowplowanalytics.snowplow/custom_event/jsonschema/1-0-0","data":{"category":"test","action":"click"}}'`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return snplog.InitLogging(cmd)
	},
}

func init() {
	EventsCmd.AddCommand(sendCmd)
}
