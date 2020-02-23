package lightsail

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer/common"
	"github.com/hashicorp/packer/helper/communicator"
	"github.com/hashicorp/packer/helper/multistep"
	"github.com/hashicorp/packer/packer"
	"log"
	"time"
)

const BuilderId = "yafimk.lightsail"

type Builder struct {
	config *Config
	runner multistep.Runner
}

func (b *Builder) ConfigSpec() hcldec.ObjectSpec { return b.config.FlatMapstructure().HCL2Spec() }

func (b *Builder) Prepare(
	raws ...interface{},
) ([]string, []string, error) {
	var err error
	b.config, err = NewConfig(raws...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed loading config: %w", err)
	}

	return nil, nil, nil
}

func (b *Builder) Run(
	ctx context.Context,
	ui packer.Ui,
	hook packer.Hook,
) (packer.Artifact, error) {

	state := new(multistep.BasicStateBag)
	state.Put("config", *b.config)
	state.Put("hook", hook)
	state.Put("ui", ui)

	staticCredentials := credentials.NewStaticCredentials(
		b.config.AccessKey,
		b.config.SecretKey,
		"")
	state.Put("creds", *staticCredentials)
	ui.Say("starting builder")
	steps := []multistep.Step{
		&StepKeyPair{DebugMode: b.config.PackerDebug, DebugKeyPath: fmt.Sprintf("ls_%s.pem",
			b.config.PackerBuildName), Comm: &b.config.Comm},
		new(StepCreateServer),
		&communicator.StepConnect{
			Config:    &b.config.Comm,
			Host:      communicator.CommHost(b.config.Comm.Host(), "server_ip"),
			SSHConfig: b.config.Comm.SSHConfigFunc(),
		},
		new(common.StepProvision),
		new(StepCreateSnapshot),
		new(StepCloneSnapshot),
		&common.StepCleanupTempKeys{Comm: &b.config.Comm},
	}

	// Run the steps
	b.runner = common.NewRunner(steps, b.config.PackerConfig, ui)

	startTime := time.Now()
	b.runner.Run(ctx, state)

	// If there was an error, return that
	if rawErr, ok := state.GetOk("error"); ok {
		return nil, rawErr.(error)
	}

	if _, ok := state.GetOk("snapshots"); !ok {
		log.Println("Failed to find snapshots in state. Bug?")
		return nil, nil
	}
	ui.Say(fmt.Sprintf("finished build flow in %f.2 min", time.Since(startTime).Minutes()))

	artifact := &Artifact{
		Name:        b.config.SnapshotName,
		RegionNames: b.config.Regions,
		creds:       staticCredentials,
	}

	return artifact, nil
}
