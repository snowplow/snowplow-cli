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
	"log/slog"
	"os"

	"github.com/snowplow-product/snowplow-cli/internal/tracking"
	"github.com/spf13/cobra"
)

var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send a single event to a Snowplow collector",
	Long: `Send a single self-describing event to a Snowplow collector.

Provide either a full self-describing JSON via --sdjson, or a --schema URI plus a
--json data payload. Optionally attach entities via --entities.`,
	Example: `  $ snowplow-cli events send -c collector.example.com -d iglu:com.snowplowanalytics.snowplow/custom_event/jsonschema/1-0-0 -j '{"category":"test","action":"click"}'
  $ snowplow-cli events send -c collector.example.com -J '{"schema":"iglu:com.snowplowanalytics.snowplow/custom_event/jsonschema/1-0-0","data":{"category":"test","action":"click"}}'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		collector, _ := cmd.Flags().GetString("collector")
		method, _ := cmd.Flags().GetString("method")
		protocol, _ := cmd.Flags().GetString("protocol")
		sdjson, _ := cmd.Flags().GetString("sdjson")
		schema, _ := cmd.Flags().GetString("schema")
		jsonData, _ := cmd.Flags().GetString("json")
		entities, _ := cmd.Flags().GetString("entities")

		appId, _ := cmd.Flags().GetString("app-id")
		if cmd.Flags().Changed("appid") && !cmd.Flags().Changed("app-id") {
			appId, _ = cmd.Flags().GetString("appid")
		}
		ipAddress, _ := cmd.Flags().GetString("ip-address")
		if cmd.Flags().Changed("ipaddress") && !cmd.Flags().Changed("ip-address") {
			ipAddress, _ = cmd.Flags().GetString("ipaddress")
		}

		code, err := tracking.Track(tracking.TrackArgs{
			Collector: collector,
			AppId:     appId,
			Method:    method,
			Protocol:  protocol,
			Sdjson:    sdjson,
			Schema:    schema,
			Json:      jsonData,
			IpAddress: ipAddress,
			Entities:  entities,
		}, nil)

		if err != nil {
			slog.Error("event send failed", "error", err)
			os.Exit(code)
		}

		slog.Debug("event sent", "collector", collector)
		return nil
	},
}

func init() {
	f := sendCmd.Flags()
	f.StringP("collector", "c", "", "Collector domain, e.g. collector.example.com (required)")
	f.StringP("app-id", "a", "snowplowcli", "Application ID")
	f.StringP("method", "m", "POST", "HTTP method [POST|GET]")
	f.StringP("protocol", "p", "https", "Protocol [http|https]")
	f.StringP("sdjson", "J", "", "Self-describing JSON of the form {\"schema\":\"iglu:...\",\"data\":{...}}")
	f.StringP("schema", "d", "", "Schema (data structure) URI, of the form iglu:...")
	f.StringP("json", "j", "", "Non-self-describing JSON data of the form {...}")
	f.StringP("ip-address", "i", "", "Custom IP address to track")
	f.StringP("entities", "e", "[]", "JSON array of self-describing JSON entities to attach")

	f.String("appid", "", "")
	_ = f.MarkDeprecated("appid", "use --app-id instead")
	f.String("ipaddress", "", "")
	_ = f.MarkDeprecated("ipaddress", "use --ip-address instead")

	_ = sendCmd.MarkFlagRequired("collector")
}
