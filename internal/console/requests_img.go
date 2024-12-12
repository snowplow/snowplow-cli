/**
 * Copyright (c) 2013-present Snowplow Analytics Ltd.
 * All rights reserved.
 * This software is made available by Snowplow Analytics, Ltd.,
 * under the terms of the Snowplow Limited Use License Agreement, Version 1.0
 * located at https://docs.snowplow.io/limited-use-license-1.0
 * BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
 * OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
 */

package console

import (
	"context"
	"io"
	"mime"
)

type Image struct {
	Ext  string
	Data []byte
}

func GetImage(cnx context.Context, client *ApiClient, path string) (*Image, error) {
	resp, err := DoConsoleRequest("GET", path, client, cnx, nil)
	if err != nil {
		return nil, err
	}
	rbody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	ext, err := mime.ExtensionsByType(resp.Header.Get("Content-Type"))

	if err != nil || ext == nil {
		return nil, err
	}

	return &Image{ext[0], rbody}, nil
}
