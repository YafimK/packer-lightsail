package lightsail

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
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
	state.Put("keyPairName", tempSSHKeyName) // default name for ssh step

	keyPairResp, err := lsClient.CreateKeyPair(&lightsail.CreateKeyPairInput{
		KeyPairName: aws.String(tempSSHKeyName),
		Tags:        nil,
	})
	if err != nil {
		err = fmt.Errorf("failed creating key pair: %w", err)
		return handleError(err, state)
	}
	var decodedPrivateKey []byte
	base64.StdEncoding.Encode(decodedPrivateKey, []byte(*keyPairResp.PrivateKeyBase64))
	privateKey, err := x509.ParsePKCS1PrivateKey(decodedPrivateKey)
	privateBlock := pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	s.Comm.SSHPrivateKey = pem.EncodeToMemory(&privateBlock)
	s.Comm.SSHUsername = tempSSHKeyName

	state.Put("privateKey", *keyPairResp) // default name for ssh step

	if s.DebugMode {
		ui.Message(fmt.Sprintf("Saving key for debug purposes: %s", s.DebugKeyPath))
		f, err := os.Create(s.DebugKeyPath)
		if err != nil {
			state.Put("error", fmt.Errorf("error saving debug key: %s", err))
			return multistep.ActionHalt
		}
		defer f.Close()
		if _, err := f.Write(pem.EncodeToMemory(&privateBlock)); err != nil {
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
	panic("implement me")
}
