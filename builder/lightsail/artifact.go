package lightsail

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lightsail"
	"log"
	"strings"
)

// Artifact represents StepCreateServer template of StepCreateServer storage as the result of StepCreateServer Packer build
type Artifact struct {
	Name        string
	RegionNames []string

	creds credentials.Credentials
}

// BuilderId returns the unique identifier of this builder
func (*Artifact) BuilderId() string {
	return BuilderId
}

// Destroy destroys the snapshot
func (a *Artifact) Destroy() error {
	log.Printf("Deleting snapshot \"%s\"", a.Name)

	awsCfg := &aws.Config{
		Credentials: &a.creds,
		Region:      &a.RegionNames[0],
	}
	newSession, err := session.NewSession(awsCfg)
	if err != nil {
		return fmt.Errorf("failed setting up aws session: %v", newSession)
	}
	lsClient := lightsail.New(newSession)

	_, err = lsClient.DeleteInstanceSnapshot(&lightsail.DeleteInstanceSnapshotInput{
		InstanceSnapshotName: aws.String(a.Name),
	})
	if err != nil {
		return fmt.Errorf("failed to delete snapshot \"%s\": %s", a.Name, err)
	}
	return nil
}

// Files returns the files represented by the artifact
func (*Artifact) Files() []string {
	return nil
}

func (a *Artifact) Id() string {
	return fmt.Sprintf("%s:%s", strings.Join(a.RegionNames[:], ","),
		a.Name)
}

func (*Artifact) State(name string) interface{} {
	return nil
}

// String returns the string representation of the artifact. It is printed at the end of builds.
func (a *Artifact) String() string {
	return fmt.Sprintf("A snapshot was created: '%v' in regions '%v'", a.Name, strings.Join(a.RegionNames[:], ","))
}
