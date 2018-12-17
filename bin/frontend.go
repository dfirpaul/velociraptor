// Frontend is not built on Windows.
package main

import (
	"net/http"

	"gopkg.in/alecthomas/kingpin.v2"
	"www.velocidex.com/golang/velociraptor/api"
	"www.velocidex.com/golang/velociraptor/gui/assets"
	"www.velocidex.com/golang/velociraptor/server"
)

var (
	// Run the server.
	frontend = app.Command("frontend", "Run the frontend and GUI.")
)

func init() {
	command_handlers = append(command_handlers, func(command string) bool {
		if command == frontend.FullCommand() {
			config_obj, err := get_server_config(*config_path)
			kingpin.FatalIfError(err, "Unable to load config file")

			server_obj, err := server.NewServer(config_obj)
			kingpin.FatalIfError(err, "Unable to create server")
			defer server_obj.Close()

			// Parse the artifacts database to detect errors early.
			getRepository(config_obj)

			// Load the assets into memory.
			assets.Init()

			go func() {
				err := api.StartServer(config_obj, server_obj)
				kingpin.FatalIfError(
					err, "Unable to start API server")
			}()

			if config_obj.AutocertDomain == "" {
				go func() {
					router := http.NewServeMux()
					err := api.PrepareMux(config_obj, router)
					kingpin.FatalIfError(
						err, "Unable to start API server")

					err = api.StartHTTPProxy(config_obj, router)
					kingpin.FatalIfError(
						err, "Unable to start GUI server")
				}()

				router := http.NewServeMux()

				// Add Comms handlers.
				server.PrepareFrontendMux(config_obj, server_obj, router)

				err = server.StartFrontendHttp(config_obj, server_obj, router)
				kingpin.FatalIfError(err, "StartFrontendHttp")
			} else {
				// For autocert we combine the GUI and
				// frontends on the same port. The
				// ACME protocol requires ports 80 and
				// 443 for all services.
				router := http.NewServeMux()
				err := api.PrepareMux(config_obj, router)
				kingpin.FatalIfError(
					err, "Unable to start API server")

				// Add Comms handlers.
				server.PrepareFrontendMux(config_obj, server_obj, router)

				// Block here until done.
				err = server.StartTLSServer(config_obj, server_obj, router)
				kingpin.FatalIfError(err, "StartTLSServer")
			}
			return true
		}
		return false
	})
}
