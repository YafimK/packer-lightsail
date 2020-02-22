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

type StepCreateServer struct{}

func (s *StepCreateServer) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {

	ui := state.Get("ui").(packer.Ui)
	config := state.Get("config").(Config)
	creds := state.Get("creds").(credentials.Credentials)
	keyPairName := state.Get("keyPairName").(string)

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

	ui.Say(fmt.Sprintf("connected to AWS region -  \"%s\" ...", config.Regions[0]))

	_, err = lsClient.CreateInstances(&lightsail.CreateInstancesInput{
		AvailabilityZone: aws.String(config.Regions[0]),
		BlueprintId:      aws.String(config.Blueprint),
		BundleId:         aws.String(config.BundleId),
		InstanceNames:    []*string{aws.String(config.SnapshotName)},
		KeyPairName:      aws.String(keyPairName),
	})
	ui.Say(fmt.Sprintf("created lightsail instance -  \"%s\" ...", config.SnapshotName))

	var lsInstance *lightsail.GetInstanceOutput
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			lsInstance, err = lsClient.GetInstance(&lightsail.GetInstanceInput{InstanceName: aws.String(config.SnapshotName)})
			if err != nil {
				err = fmt.Errorf("failed creating instance: %w", err)
				return handleError(err, state)
			}
			state.Put("server_details", *lsInstance)
			if *lsInstance.Instance.State.Code != 16 {
				continue
			}
			break
		case <-ctx.Done():
			ticker.Stop()
			err = fmt.Errorf("failed creating instance: %w", err)
			return handleError(ctx.Err(), state)
		}
		break
	}

	state.Put("server_details", *lsInstance)
	state.Put("server_ip", *lsInstance.Instance.PublicIpAddress)

	ui.Say(fmt.Sprintf("Deployed snapshot instance \"%s\" is now \"active\" state", *lsInstance.Instance.Name))

	return multistep.ActionContinue
}

func (s *StepCreateServer) Cleanup(state multistep.StateBag) {
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
