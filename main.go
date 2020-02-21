package main

import (
	"github.com/hashicorp/packer/packer/plugin"
	"github.com/yafimk/lightsail-packer/builder/lightsail"
)

func main() {
	server, err := plugin.Server()
	if err != nil {
		panic(err)
	}
	server.RegisterBuilder(new(lightsail.Builder))
	server.Serve()
}



