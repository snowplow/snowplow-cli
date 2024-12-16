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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/snowplow-product/snowplow-cli/internal/util"
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

type imageLookup struct {
	Lookup map[string]string
}

type ImageHashes = []string

func GetImageHashLookup(cnx context.Context, client *ApiClient) (ImageHashes, error) {
	resp, err := DoConsoleRequest("GET", fmt.Sprintf("%s/images/v1/hash-lookup", client.BaseUrl), client, cnx, nil)
	if err != nil {
		return nil, err
	}
	rbody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var lookup imageLookup
	err = json.Unmarshal(rbody, &lookup)
	if err != nil {
		return nil, err
	}

	hashes := []string{}
	for _, v := range lookup.Lookup {
		if v != "" {
			hashes = append(hashes, v)
		}
	}

	return hashes, nil
}

type imageUploadLinkResponse struct {
	UploadURL string
	Id string
}

func createImageUploadLink(cnx context.Context, client *ApiClient) (*imageUploadLinkResponse, error) {
	resp, err := DoConsoleRequest("POST", fmt.Sprintf("%s/images/v1", client.BaseUrl), client, cnx, nil)
	if err != nil {
		return nil, err
	}
	rbody, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("create upload link failure")
	}

	var createResp imageUploadLinkResponse
	err = json.Unmarshal(rbody, &createResp)
	if err != nil {
		return nil, err
	}

	return &createResp, nil
}

func uploadImage(cnx context.Context, client *ApiClient, fname string, uploadLink *imageUploadLinkResponse) error {
	upBuf := bytes.Buffer{}
	writer := multipart.NewWriter(&upBuf)
	cfFilename := fmt.Sprintf(
		"%s_%s_%s",
		client.OrgId,
		uploadLink.Id,
		util.ResourceNameToFileName(filepath.Base(fname)),
	)
	part, err := writer.CreateFormFile("file", cfFilename)
	if err != nil {
		return err
	}

	f, err := os.Open(fname)
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	_, err = part.Write(b)
	if err != nil {
		return err
	}

	writer.Close()

	req, err := http.NewRequestWithContext(cnx, "POST", uploadLink.UploadURL, &upBuf)
	if err != nil {
		return err
	}
	req.Header.Set("content-type", writer.FormDataContentType())
	resp, err := client.Http.Do(req)
	if err != nil {
		return err
	}
	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	var cfResp struct{ Success bool; Errors []struct{ Code int; Message string } }
	err = json.Unmarshal(body, &cfResp)
	if err != nil {
		return err
	}

	if !cfResp.Success {
		cfErr, err := json.Marshal(cfResp.Errors)
		if err != nil {
			return err
		}
		return fmt.Errorf("upload failed file:%s cfError: %s", fname, string(cfErr))
	}

	return nil
}

type VariantUrls map[string]string

func confirmImage(cnx context.Context, client *ApiClient, id string, hash string) (VariantUrls, error) {
	req, err := json.Marshal(map[string]string{ "hash": hash })
	if err != nil {
		return nil, err
	}

	resp, err := DoConsoleRequest("POST",
		fmt.Sprintf("%s/images/v1/%s/confirm-upload", client.BaseUrl, id),
		client, cnx, bytes.NewBuffer(req),
	)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}

	var confirmResp struct{ VariantUrls map[string]string }
	err = json.Unmarshal(body, &confirmResp)
	if err != nil {
		return nil, err
	}

	return confirmResp.VariantUrls, nil
}

func PublishImage(cnx context.Context, client *ApiClient, fname string, hash string) (VariantUrls, error) {
	upload, err := createImageUploadLink(cnx, client)
	if err != nil {
		return nil, err
	}

	err = uploadImage(cnx, client, fname, upload)
	if err != nil {
		return nil, err
	}

	return confirmImage(cnx, client, upload.Id, hash)
}
