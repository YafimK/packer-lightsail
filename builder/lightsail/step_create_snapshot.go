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

type StepCreateSnapshot struct{}

func (s *StepCreateSnapshot) Run(
	ctx context.Context,
	state multistep.StateBag,
) multistep.StepAction {

	ui := state.Get("ui").(packer.Ui)
	config := state.Get("config").(Config)
	creds := state.Get("creds").(credentials.Credentials)
	serverDetails := state.Get("server_details").(lightsail.GetInstanceOutput)
	awsCfg := &aws.Config{
		Credentials: &creds,
		Region:      aws.String(getCentralRegion(config.Regions[0])),
	}

	newSession, err := session.NewSession(awsCfg)
	if err != nil {
		err = fmt.Errorf("failed setting up aws session: %v", newSession)
		return handleError(err, state)
	}
	lsClient := lightsail.New(newSession)
	ui.Say(fmt.Sprintf("creating snapshot \"%s\" in  \"%s\" ..", config.SnapshotName, config.Regions[0]))

	_, err = lsClient.CreateInstanceSnapshot(&lightsail.CreateInstanceSnapshotInput{
		InstanceName:         serverDetails.Instance.Name,
		InstanceSnapshotName: aws.String(config.SnapshotName),
	})
	if err != nil {
		err = fmt.Errorf("failed creating snapshot: %w", err)
		return handleError(err, state)
	}
	ui.Say(fmt.Sprintf("finished creating snapshot \"%s\" in  \"%s\" ...", config.SnapshotName, config.Regions[0]))
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
			state.Put("snapshot_details", *snapshot.InstanceSnapshot)
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

	state.Put("snapshot_details", *snapshot.InstanceSnapshot)
	ui.Say(fmt.Sprintf("Deployed snapshot \"%s\" is now in \"%s\" state", *snapshot.InstanceSnapshot.Name,
		*snapshot.InstanceSnapshot.State))
	return multistep.ActionContinue

}

func (s *StepCreateSnapshot) Cleanup(state multistep.StateBag) {
}
