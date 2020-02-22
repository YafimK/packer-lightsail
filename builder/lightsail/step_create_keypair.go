package lightsail

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/hashicorp/packer/common/uuid"
	"github.com/hashicorp/packer/helper/communicator"
	"github.com/hashicorp/packer/helper/multistep"
	"github.com/hashicorp/packer/packer"
	"os"
	"runtime"
)

type StepKeyPair struct {
	DebugMode    bool
	DebugKeyPath string
	Comm         *communicator.Config
}

func (s *StepKeyPair) Run(
	ctx context.Context,
	state multistep.StateBag,
) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)
	config := state.Get("config").(Config)
	creds := state.Get("creds").(credentials.Credentials)

	awsRegion := getCentralRegion(config.Regions[0])
	awsCfg := &aws.Config{
		Credentials: &creds,
		Region:      aws.String(awsRegion),
	}
	newSession, err := session.NewSession(awsCfg)
	if err != nil {
		err = fmt.Errorf("failed setting up aws session: %v", newSession)
		return handleError(err, state)
	}
	lsClient := lightsail.New(newSession)
	ui.Say(fmt.Sprintf("connected to AWS region -  \"%s\" ...", awsRegion))

	tempSSHKeyName := fmt.Sprintf("packer-%s", uuid.TimeOrderedUUID())
	state.Put("keyPairName", tempSSHKeyName) // default name for ssh step
	keyPairResp, err := lsClient.CreateKeyPair(&lightsail.CreateKeyPairInput{
		KeyPairName: aws.String(tempSSHKeyName),
		Tags:        nil,
	})
	if err != nil {
		err = fmt.Errorf("failed creating key pair: %w", err)
		return handleError(err, state)
	}
	decodedPrivateKey := []byte(*keyPairResp.PrivateKeyBase64)
	s.Comm.SSHPrivateKey = decodedPrivateKey
	s.Comm.SSHPublicKey = []byte(*keyPairResp.PublicKeyBase64)
	s.Comm.SSHUsername = "ubunutu"

	state.Put("keypair", *keyPairResp) // default name for ssh step

	if s.DebugMode {
		ui.Message(fmt.Sprintf("Saving key for debug purposes: %s", s.DebugKeyPath))
		f, err := os.Create(s.DebugKeyPath)
		if err != nil {
			state.Put("error", fmt.Errorf("error saving debug key: %s", err))
			return multistep.ActionHalt
		}
		defer f.Close()
		if _, err := f.Write(decodedPrivateKey); err != nil {
			state.Put("error", fmt.Errorf("error saving debug key: %s", err))
			return multistep.ActionHalt
		}

		if runtime.GOOS != "windows" {
			if err := f.Chmod(0600); err != nil {
				state.Put("error", fmt.Errorf("error setting permissions of debug key: %s", err))
				return multistep.ActionHalt
			}
		}
	}

	return multistep.ActionContinue

}

func (s *StepKeyPair) Cleanup(multistep.StateBag) {
}
