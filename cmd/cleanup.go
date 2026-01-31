package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/linode/linodego"
	"golang.org/x/oauth2"
)

// Use this file for cloud provider specific cleanup funcs, or anything that
// for whatever reason Pulumi is not able to manage/destroy after deployment.

type LinodeLogger struct {
	log *slog.Logger
}

func (l *LinodeLogger) Errorf(format string, v ...any) {
	l.log.Error(fmt.Sprintf(format, v...))
}

func (l *LinodeLogger) Warnf(format string, v ...any) {
	l.log.Warn(fmt.Sprintf(format, v...))
}

func (l *LinodeLogger) Debugf(format string, v ...any) {
	l.log.Debug(fmt.Sprintf(format, v...))
}

// purgeLkeClusterResources deletes an entire LKE cluster and any leftover cloud
// resources such as Block Storage Volumes or NodeBalancers.
func purgeLkeClusterResources(ctx context.Context, lkeid int) {
	// setup client
	client := NewLinodeClient()
	volumeAttachedRetryCondition := func(r *resty.Response, err error) bool {
		return r.StatusCode() == 400
	}
	client.AddRetryCondition(volumeAttachedRetryCondition)
	client.SetRetryCount(5).SetRetryMaxWaitTime(10 * time.Second)

	// create a list of volume ids
	// note: this has to be done before deleting the cluster
	label := fmt.Sprintf("lke%d", lkeid)

	volumes, err := client.ListVolumes(ctx, &linodego.ListOptions{})
	if err != nil {
		logger.Error("list volumes: " + err.Error())
	}

	volIds := make([]int, 0)

	for _, i := range volumes {
		if strings.Contains(i.LinodeLabel, label) {
			volIds = append(volIds, i.ID)
		}
	}

	// tag volumes in case we need to rebuild list after lke cluster is deleted
	tag := platform.Name + "-volume"

	if len(volIds) > 0 {
		for _, i := range volIds {
			_, err := client.UpdateVolume(ctx, i, linodego.VolumeUpdateOptions{
				Tags: &[]string{tag},
			})
			if err != nil {
				logger.Error("tagging linode volumes: " + err.Error())
			}
		}
	} else {
		for _, i := range volumes {
			t := i.Tags
			if slices.Contains(t, tag) {
				volIds = append(volIds, i.ID)
			}
		}
	}

	// delete lke cluster
	if err := client.DeleteLKECluster(ctx, lkeid); err != nil {
		if !strings.Contains(err.Error(), "Not found") {
			logger.Error("delete lke cluster: " + err.Error())
		}

		logger.Info("purged lke cluster")
		time.Sleep(10 * time.Second)
	}

	// delete volumes
	for idx, i := range volIds {
		idx++
		msg := fmt.Sprintf("(%d/%d) purging volumes", idx, len(volIds))
		logger.Info(msg)

		if err := client.DetachVolume(ctx, i); err != nil {
			logger.Error("detach volume: " + err.Error())
		}

		time.Sleep(5 * time.Second)

		if err := client.DeleteVolume(ctx, i); err != nil {
			logger.Error("delete volume: " + err.Error())
		}
	}

	// delete nodebalancers
	deleteNodeBalancers(ctx, lkeid)
}

func deleteNodeBalancers(ctx context.Context, lkeid int) {
	client := NewLinodeClient()
	label := fmt.Sprintf("lke%d", lkeid)

	nodebalancers, err := client.ListNodeBalancers(ctx, &linodego.ListOptions{})
	if err != nil {
		logger.Error("list nodebalancers: " + err.Error())
	}

	for _, i := range nodebalancers {
		if strings.Contains(*i.Label, label) {
			err := client.DeleteNodeBalancer(ctx, i.ID)
			if err != nil {
				logger.Error("delete nodebalancers: " + err.Error())
			}
		}
	}

	logger.Info("purged nodebalancers")
}

func NewLinodeClient() linodego.Client {
	apikey, ok := os.LookupEnv("LINODE_TOKEN")
	if !ok {
		logger.Error("new linode client: could not find LINODE_TOKEN")
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: apikey})
	oauth2Client := &http.Client{
		Transport: &oauth2.Transport{
			Source: tokenSource,
		},
	}

	client := linodego.NewClient(oauth2Client)
	client.SetLogger(&LinodeLogger{log: logger})

	return client
}
