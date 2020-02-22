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

	_, err = lsClient.CreateInstanceSnapshot(&lightsail.CreateInstanceSnapshotInput{
		InstanceName:         aws.String(config.SnapshotName),
		InstanceSnapshotName: aws.String(config.SnapshotName),
		Tags:                 nil,
	})
	if err != nil {
		return handleError(err, state)
	}

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
	ui.Say(fmt.Sprintf("Deployed snapshot \"%s\" is now \"active\" state", *snapshot.InstanceSnapshot.Name))

	serverDetails := state.Get("server_details").(lightsail.GetInstanceOutput)

	ui.Say(fmt.Sprintf("Deleting server \"%s\" ...", serverDetails))

	_, err = lsClient.DeleteInstance(&lightsail.DeleteInstanceInput{
		ForceDeleteAddOns: nil,
		InstanceName:      aws.String(config.SnapshotName),
	})
	if err != nil {
		return handleError(fmt.Errorf("failed to delete server \"%s\": %s", *serverDetails.Instance.Name, err), state)
	}
	return multistep.ActionContinue

}

func (s *StepCreateSnapshot) Cleanup(state multistep.StateBag) {

	rawDetails, isExist := state.GetOk("server_details")
	if !isExist {
		return
	}
	serverDetails := rawDetails.(lightsail.GetInstanceOutput)

	ui := state.Get("ui").(packer.Ui)
	config := state.Get("config").(Config)
	creds := state.Get("creds").(credentials.Credentials)

	awsCfg := &aws.Config{
		Credentials: &creds,
		Region:      aws.String(getCentralRegion(config.Regions[0])),
	}
	newSession, err := session.NewSession(awsCfg)
	if err != nil {
		ui.Say(fmt.Sprintf("failed setting up aws session: %v", newSession))
		return
	}
	lsClient := lightsail.New(newSession)
	ui.Say(fmt.Sprintf("Deleting server \"%s\" ...", serverDetails))

	_, err = lsClient.DeleteInstance(&lightsail.DeleteInstanceInput{
		ForceDeleteAddOns: nil,
		InstanceName:      aws.String(config.SnapshotName),
	})
	if err != nil {
		ui.Error(fmt.Sprintf("failed to delete server \"%s\": %s", *serverDetails.Instance.Name, err))
	}
}
