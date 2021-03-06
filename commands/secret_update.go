// Copyright (c) OpenFaaS Author(s) 2019. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package commands

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/openfaas/faas-cli/proxy"
	"github.com/openfaas/faas-cli/schema"
	"github.com/spf13/cobra"
)

var secretUpdateCmd = &cobra.Command{
	Use:     "update [--tls-no-verify]",
	Aliases: []string{"u"},
	Short:   "Update a secret",
	Long:    `Update a secret by name`,
	Example: `faas-cli secret update NAME
faas-cli secret update NAME --from-literal=secret-value
faas-cli secret update NAME --from-file=/path/to/secret/file
faas-cli secret update NAME --from-literal=secret-value --gateway=http://127.0.0.1:8080
cat /path/to/secret/file | faas-cli secret update NAME`,
	RunE:    runSecretUpdate,
	PreRunE: preRunSecretUpdate,
}

func init() {
	secretUpdateCmd.Flags().StringVarP(&gateway, "gateway", "g", defaultGateway, "Gateway URL starting with http(s)://")
	secretUpdateCmd.Flags().BoolVar(&tlsInsecure, "tls-no-verify", false, "Disable TLS validation")
	secretUpdateCmd.Flags().StringVar(&literalSecret, "from-literal", "", "Value of the secret")
	secretUpdateCmd.Flags().StringVar(&secretFile, "from-file", "", "Path to the secret file")
	secretUpdateCmd.Flags().StringVarP(&token, "token", "k", "", "Pass a JWT token to use instead of basic auth")
	secretCmd.AddCommand(secretUpdateCmd)
}

func preRunSecretUpdate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("secret name required")
	}

	if len(args) > 1 {
		return fmt.Errorf("too many values for secret name")
	}

	if len(secretFile) > 0 && len(literalSecret) > 0 {
		return fmt.Errorf("please provide secret using only one option from --from-literal, --from-file and STDIN")
	}

	return nil
}

func runSecretUpdate(cmd *cobra.Command, args []string) error {
	gatewayAddress := getGatewayURL(gateway, defaultGateway, "", os.Getenv(openFaaSURLEnvironment))

	if msg := checkTLSInsecure(gatewayAddress, tlsInsecure); len(msg) > 0 {
		fmt.Println(msg)
	}

	secret := schema.Secret{
		Name: args[0],
	}

	switch {
	case len(literalSecret) > 0:
		secret.Value = literalSecret

	case len(secretFile) > 0:
		content, err := ioutil.ReadFile(secretFile)
		if err != nil {
			return fmt.Errorf("unable to read secret file: %s", err.Error())
		}
		secret.Value = string(content)

	default:
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			fmt.Fprintf(os.Stderr, "Reading from STDIN - hit (Control + D) to stop.\n")
		}

		secretStdin, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("unable to read standard input: %s", err.Error())
		}
		secret.Value = string(secretStdin)
	}

	secret.Value = strings.TrimSpace(secret.Value)

	if len(secret.Value) == 0 {
		return fmt.Errorf("must provide a non empty secret via --from-literal, --from-file or STDIN")
	}

	fmt.Println("Updating secret: " + secret.Name)
	_, output := proxy.UpdateSecretToken(gatewayAddress, secret, tlsInsecure, token)
	fmt.Printf(output)

	return nil
}
