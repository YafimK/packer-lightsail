package lightsail

import (
	"github.com/hashicorp/packer/common"
	"github.com/hashicorp/packer/helper/config"
	"github.com/hashicorp/packer/template/interpolate"
	"time"
)

type Config struct {
	common.PackerConfig `mapstructure:",squash"`

	ctx             interpolate.Context
	PackerBuildName string

	SnapshotName string   `mapstructure:"snapshot_name"`
	Regions      []string `mapstructure:"regions"`
	BundleId     string   `mapstructure:"bundle_id"`
	Blueprint    string   `mapstructure:"blueprint_id"`

	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`

	PublicKeyUser string `mapstructure:"ssh_user"`

	Timeout time.Duration `mapstructure:"timeout"`
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
}
