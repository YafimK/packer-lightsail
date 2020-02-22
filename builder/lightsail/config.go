package lightsail

import (
	"github.com/hashicorp/packer/common"
	"github.com/hashicorp/packer/helper/communicator"
	"github.com/hashicorp/packer/helper/config"
	"github.com/hashicorp/packer/template/interpolate"
	"time"
)

type Config struct {
	common.PackerConfig `mapstructure:",squash"`

	ctx             interpolate.Context
	PackerBuildName string

	SnapshotName string   `mapstructure:"snapshot_name" required:"true"`
	Regions      []string `mapstructure:"regions" required:"true"`
	BundleId     string   `mapstructure:"bundle_id" required:"true"`
	Blueprint    string   `mapstructure:"blueprint_id" required:"true"`

	AccessKey string `mapstructure:"access_key" required:"true"`
	SecretKey string `mapstructure:"secret_key" required:"true"`

	Timeout time.Duration `mapstructure:"timeout"`

	Comm  communicator.Config `mapstructure:",squash"`
	Debug bool                `mapstructure:"debug"`
}

func NewConfig(raws ...interface{}) (*Config, error) {
	c := new(Config)

	err := config.Decode(c, &config.DecodeOpts{
		Interpolate:        true,
		InterpolateContext: &c.ctx,
	}, raws...)

	if err != nil {
		return nil, err
	}

	return c, nil
}
