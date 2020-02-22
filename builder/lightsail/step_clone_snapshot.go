package lightsail

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/hashicorp/packer/helper/multistep"
	"github.com/hashicorp/packer/packer"
	"golang.org/x/sync/errgroup"
	"time"
)

type StepCloneSnapshot struct{}

func (s *StepCloneSnapshot) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {

	ui := state.Get("ui").(packer.Ui)
	config := state.Get("config").(Config)
	creds := state.Get("creds").(credentials.Credentials)
	snapshot := state.Get("snapshot_details").(lightsail.InstanceSnapshot)

	if len(config.Regions) < 1 {
		return multistep.ActionContinue
	}
	ui.Say(fmt.Sprintf("Deploying snapshot \"%s\" into regions: %v", *snapshot.Name, config.Regions[1:]))
	errgrp, _ := errgroup.WithContext(ctx)
	var snapshots []lightsail.InstanceSnapshot
	for _, region := range config.Regions[1:] {
		region := region
		errgrp.Go(func() error {
			awsRegion := getCentralRegion(region)
			awsCfg := &aws.Config{
				Credentials: &creds,
				Region:      aws.String(awsRegion),
			}
			newSession, err := session.NewSession(awsCfg)
			if err != nil {
				err := fmt.Errorf("failed setting up aws session: %v", newSession)
				return err
			}
			lsClient := lightsail.New(newSession)

			ui.Say(fmt.Sprintf("connected to AWS region -  \"%s\" ...", awsRegion))
			ui.Say(fmt.Sprintf("creating snapshot \"%s\" in  \"%s\" ..", config.SnapshotName, region))
			_, err = lsClient.CopySnapshot(&lightsail.CopySnapshotInput{
				SourceRegion:       snapshot.Location.RegionName,
				SourceSnapshotName: snapshot.Name,
				TargetSnapshotName: snapshot.Name,
			})
			if err != nil {
				err = fmt.Errorf("failed cloning snapshot: %w", err)
				return err
			}
			ui.Say(fmt.Sprintf("waiting for snapshot \"%s\" in  \"%s\" ..", config.SnapshotName, region))
			var snapshot *lightsail.GetInstanceSnapshotOutput
			ticker := time.NewTicker(5 * time.Second)
			for {
				select {
				case <-ticker.C:
					snapshot, err = lsClient.GetInstanceSnapshot(&lightsail.GetInstanceSnapshotInput{InstanceSnapshotName: aws.
						String(config.SnapshotName)})
					if err != nil {
						return err
					}
					if *snapshot.InstanceSnapshot.State != lightsail.InstanceSnapshotStateAvailable {
						continue
					}
					break
				case <-ctx.Done():
					ticker.Stop()
					return ctx.Err()
				}
				break
			}
			snapshots = append(snapshots, *snapshot.InstanceSnapshot)
			ui.Say(fmt.Sprintf("Deployed snapshot \"%s\" is now in \"%s\" state", *snapshot.InstanceSnapshot.Name,
				*snapshot.InstanceSnapshot.State))
			return nil
		})
	}
	if err := errgrp.Wait(); err != nil {
		return handleError(err, state)
	}

	state.Put("snapshots", snapshots)

	return multistep.ActionContinue

}

func (s *StepCloneSnapshot) Cleanup(state multistep.StateBag) {
}
