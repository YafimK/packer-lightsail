package lightsail

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/hashicorp/packer/common/uuid"
	"github.com/hashicorp/packer/helper/communicator"
	"github.com/hashicorp/packer/helper/multistep"
	"github.com/hashicorp/packer/packer"
)

type StepKeyPair struct {
	Comm *communicator.Config
}

func (s *StepKeyPair) Run(
	ctx context.Context,
	state multistep.StateBag,
) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)
	config := state.Get("config").(Config)
	creds := state.Get("creds").(credentials.Credentials)

	awsCfg := &aws.Config{
		Credentials: &creds,
		Region:      &config.Regions[0],
	}
	newSession, err := session.NewSession(awsCfg)
	if err != nil {
		err = fmt.Errorf("failed setting up aws session: %v", newSession)
		return handleError(err, state)
	}
	lsClient := lightsail.New(newSession)

	ui.Say(fmt.Sprintf("connected to AWS region -  \"%s\" ...", config.Regions[0]))
	tempSSHKeyName := fmt.Sprintf("packer-%s", uuid.TimeOrderedUUID())

	keyPairResp, err := lsClient.CreateKeyPair(&lightsail.CreateKeyPairInput{
		KeyPairName: aws.String(tempSSHKeyName),
		Tags:        nil,
	})
	if err != nil {
		return handleError(err, state)
	}
	var decodedPrivateKey []byte
	base64.StdEncoding.Encode(decodedPrivateKey, []byte(*keyPairResp.PrivateKeyBase64))
	s.Comm.SSHPrivateKey = decodedPrivateKey
	state.Put("sshKey", *keyPairResp)

	return multistep.ActionContinue

}

func (s *StepKeyPair) Cleanup(multistep.StateBag) {
	panic("implement me")
}
