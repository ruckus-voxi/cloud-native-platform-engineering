package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/rclone/rclone/backend/all"
	_ "github.com/rclone/rclone/fs/operations" // import operations/*
	_ "github.com/rclone/rclone/fs/sync"       // import sync/*
	"github.com/rclone/rclone/librclone/librclone"
)

// This file is used by the aplicli tool that generated this project, not by the
// project itself, but is included here as a template or starting point for
// future rclone operations a user may want to implement.

//nolint:unused
type syncRequest struct {
	SrcFs     string `json:"srcFs"`
	DstFs     string `json:"dstFs"`
	DstRemote string `json:"dstRemote"`
	Group     string `json:"_group"`
	Async     bool   `json:"_async"`
}

type PurgeRequest struct {
	Fs     string `json:"fs"`
	Remote string `json:"remote"`
	RmDirs bool   `json:"rmdirs,omitempty"` // add --rmDirs flag to rclone delete command
}

type s3Remote struct {
	AccessKeyId     string            `json:"accessKey,omitempty"`
	Acl             string            `json:"acl,omitempty"`
	Buckets         map[string]string `json:"objBuckets,omitempty"`
	Endpoint        string            `json:"endpoint,omitempty"`
	Provider        string            `json:"provider,omitempty"`
	PurgeEnabled    bool              `json:"purge,omitempty"`
	Remote          string            `json:"name,omitempty"`
	SecretAccessKey string            `json:"secretKey,omitempty"`
}

func (r *s3Remote) Init(ctx context.Context) {
	required := []string{
		r.AccessKeyId,
		r.SecretAccessKey,
		r.Endpoint,
		r.Remote,
	}

	for _, i := range required {
		if i == "" {
			txt := "AccessKeyId, SecretAccessKey, Endpoint, Remote"
			msg := fmt.Sprintf("missing one or more requied field values(%s)", txt)
			logger.Error("initialize s3Remote: " + msg)
		}
	}

	if r.Provider == "" {
		r.Provider = "Other"
	}

	if r.Acl == "" {
		r.Acl = "private"
	}
	// https://rclone.org/s3/#magalu
	envVars := map[string]string{
		"RCLONE_S3_ACCESS_KEY_ID":     r.AccessKeyId,
		"RCLONE_S3_ACL":               r.Acl,
		"RCLONE_S3_ENDPOINT":          r.Endpoint,
		"RCLONE_S3_ENV_AUTH":          "true",
		"RCLONE_S3_PROVIDER":          r.Provider,
		"RCLONE_S3_SECRET_ACCESS_KEY": r.SecretAccessKey,
	}
	for k, v := range envVars {
		if err := os.Setenv(k, v); err != nil {
			msg := fmt.Sprintf("set `%s=%s` rclone environment variable", k, v)
			logger.Error(msg + err.Error())
		}
	}
}

func (r *s3Remote) Purge(ctx context.Context, bkt ...string) {
	methods := map[string]string{
		"list":   "operations/list",
		"delete": "operations/delete",
		"purge":  "operations/purge",
	}

	buckets := make([]string, 0)
	if len(bkt) > 0 {
		buckets = bkt
	} else {
		for _, v := range r.Buckets {
			buckets = append(buckets, v)
		}
	}

	for idx, bucket := range buckets {
		var method string

		idx++
		f := ":s3:" + bucket
		purgeReq := PurgeRequest{
			// To use without ENV_AUTH, the string value for FS needs to be
			// formatted like this:
			// ":s3,provider=Other,env_auth=false,access_key_id=<ACCESS_KEY>,secret_access_key=<SECRET_ACCESS_KEY>,endpoint='nl-ams-1.linodeobjects.com':<BUCKET>,
			Fs:     f,
			Remote: bucket,
		}

		// skip bucket if does not exist
		method = methods["list"]

		requestJSON, err := json.Marshal(purgeReq)
		if err != nil {
			logger.Error("json marshal rclone bucket list request: " + err.Error())
		}

		res, status := rcloneAction(method, string(requestJSON))
		if status != 200 {
			if status == 404 {
				msg := fmt.Sprintf("(%d/%d) skipping bucket %s", idx, len(buckets), bucket)
				logger.Info(msg)

				continue
			} else {
				msg := fmt.Sprintf("list bucket operation: status %v, response: %v", status, res)
				logger.Error(msg)
			}
		}

		if !r.PurgeEnabled {
			purgeReq.RmDirs = true
			method = methods["delete"]
		} else {
			method = methods["purge"]
		}

		requestJSON, err = json.Marshal(purgeReq)
		if err != nil {
			logger.Error("json marshal rclone purge/delete request: " + err.Error())
		}

		msg := fmt.Sprintf("(%d/%d) deleting objects in bucket %s", idx, len(buckets), bucket)
		logger.Info(msg)

		res, status = rcloneAction(method, string(requestJSON))
		if status != 200 {
			msg := fmt.Sprintf("delete bucket operation status %v, response: %v", status, res)
			logger.Error(msg)
		}
	}
}

func rcloneAction(method, req string) (string, int) {
	librclone.Initialize()
	defer librclone.Finalize()

	res, status := librclone.RPC(method, req)

	return res, status
}
