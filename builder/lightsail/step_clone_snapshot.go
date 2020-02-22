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

	var snapshots []lightsail.InstanceSnapshot
	for _, region := range config.Regions[1:] {
		awsRegion := getCentralRegion(region)
		awsCfg := &aws.Config{
			Credentials: &creds,
			Region:      aws.String(awsRegion),
		}
		newSession, err := session.NewSession(awsCfg)
		if err != nil {
			err := fmt.Errorf("failed setting up aws session: %v", newSession)
			return handleError(err, state)
		}
		lsClient := lightsail.New(newSession)

		ui.Say(fmt.Sprintf("connected to AWS region -  \"%s\" ...", awsRegion))
		ui.Say(fmt.Sprintf("creating snapshot \"%s\" in  \"%s\" ..", config.SnapshotName, config.Regions[0]))
		_, err = lsClient.CopySnapshot(&lightsail.CopySnapshotInput{
			SourceRegion:       snapshot.Location.RegionName,
			SourceSnapshotName: snapshot.Name,
			TargetSnapshotName: snapshot.Name,
		})
		if err != nil {
			err = fmt.Errorf("failed cloning snapshot: %w", err)
			return handleError(err, state)
		}
		ui.Say(fmt.Sprintf("waiting for snapshot \"%s\" to be ready", config.SnapshotName))
		var snapshot *lightsail.GetInstanceSnapshotOutput
		ticker := time.NewTicker(5 * time.Second)
		for {
			select {
			case <-ticker.C:
				snapshot, err = lsClient.GetInstanceSnapshot(&lightsail.GetInstanceSnapshotInput{InstanceSnapshotName: aws.
					String(config.SnapshotName)})
				if err != nil {
					return handleError(err, state)
				}
				if *snapshot.InstanceSnapshot.State != lightsail.InstanceSnapshotStateAvailable {
					continue
				}
				break
			case <-ctx.Done():
				ticker.Stop()
				return handleError(ctx.Err(), state)
			}
			break
		}
		snapshots = append(snapshots, *snapshot.InstanceSnapshot)
		ui.Say(fmt.Sprintf("Deployed snapshot \"%s\" is now in \"%s\" state", *snapshot.InstanceSnapshot.Name,
			*snapshot.InstanceSnapshot.State))
	}

	state.Put("snapshots", snapshots)

	return multistep.ActionContinue

}

func (s *StepCloneSnapshot) Cleanup(state multistep.StateBag) {
}
