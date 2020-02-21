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

	for _, region := range config.Regions {
		awsCfg := &aws.Config{
			Credentials: &creds,
			Region:      &region,
		}
		newSession, err := session.NewSession(awsCfg)
		if err != nil {
			err := fmt.Errorf("failed setting up aws session: %v", newSession)
			return handleError(err, state)
		}
		lsClient := lightsail.New(newSession)

		ui.Say(fmt.Sprintf("connected to AWS region -  \"%s\" ...", config.Regions[0]))

		_, err = lsClient.CopySnapshot(&lightsail.CopySnapshotInput{
			SourceRegion:       &config.Regions[0],
			SourceSnapshotName: snapshot.Name,
			TargetSnapshotName: snapshot.Name,
		})
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

		ui.Say(fmt.Sprintf("Deployed snapshot \"%s\" is now \"active\" state", *snapshot.InstanceSnapshot.Name))

	}

	state.Put("snapshot_details", *snapshot.InstanceSnapshot)

	return multistep.ActionContinue

}

func (s *StepCloneSnapshot) Cleanup(state multistep.StateBag) {
	panic("implement me")
}
