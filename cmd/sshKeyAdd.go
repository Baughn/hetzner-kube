// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"

	"errors"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

// sshKeyAddCmd represents the sshKeyAdd command
var sshKeyAddCmd = &cobra.Command{
	Use:   "add",
	Short: "adds a new SSH key to the Hetzner Cloud project and local configuration",
	Long: `This sub-command saves the path of the provided SSH private key in a configuration file on your local machine.
Then it uploads it corresponding public key with the provided name to the Hetzner Cloud project, associated by the current context.

Note: the private key is never uploaded to any server at any time.`,
	PreRunE: validateFlags,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("sshKeyAdd called")
		name, _ := cmd.Flags().GetString("name")
		publicKeyPath, _ := cmd.Flags().GetString("public-key-path")
		privateKeyPath, _ := cmd.Flags().GetString("private-key-path")
		generate, _ := cmd.Flags().GetBool("generate")

		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		privateKeyPath = strings.Replace(privateKeyPath, "~", home, 1)
		publicKeyPath = strings.Replace(publicKeyPath, "~", home, 1)

		var (
			data []byte
			publicKey string
		)
		if generate {
			if pubk, pk, err := GenKeyPair(); err == nil {
				ioutil.WriteFile(privateKeyPath, []byte(pk), 0600)
				ioutil.WriteFile(publicKeyPath, []byte(pubk), 0644)
				publicKey = pubk
			} else {
				log.Fatal(err)
			}
		} else {
			if publicKeyPath == "-" {
				data, err = ioutil.ReadAll(os.Stdin)
			} else {
				data, err = ioutil.ReadFile(publicKeyPath)
			}
			FatalOnError(err)
			publicKey = string(data)
		}

		opts := hcloud.SSHKeyCreateOpts{
			Name:      name,
			PublicKey: publicKey,
		}

		context := AppConf.Context
		client := AppConf.Client
		fmt.Println(publicKey)
		sshKey, _, err := client.SSHKey.Create(context, opts)

		if err != nil {
			log.Fatalln(err)
		}

		AppConf.Config.AddSSHKey(SSHKey{
			Name:           name,
			PrivateKeyPath: privateKeyPath,
			PublicKeyPath:  publicKeyPath,
		})

		AppConf.Config.WriteCurrentConfig()

		fmt.Printf("SSH key %s(%d) created\n", name, sshKey.ID)
	},
}

func validateFlags(cmd *cobra.Command, args []string) error {
	if err := AppConf.assertActiveContext(); err != nil {
		return err
	}

	if name, _ := cmd.Flags().GetString("name"); name == "" {
		return errors.New("flag --name is required")
	}

	generate, _ := cmd.Flags().GetBool("generate")

	privateKeyPath, _ := cmd.Flags().GetString("private-key-path")
	if privateKeyPath == "" {
		return errors.New("flag --private-key-path cannot be empty")
	}

	publicKeyPath, _ := cmd.Flags().GetString("public-key-path")
	if publicKeyPath == "" {
		return errors.New("flag --public-key-path cannot be empty")
	}

	// Find home directory.
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	privateKeyPath = strings.Replace(privateKeyPath, "~", home, 1)
	publicKeyPath = strings.Replace(publicKeyPath, "~", home, 1)
	if !generate {
		if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
			return errors.New(fmt.Sprintf("could not find private key '%s'", privateKeyPath))

		}

		if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
			return errors.New(fmt.Sprintf("could not find public key '%s'", publicKeyPath))
		}
	}

	return nil
}

func init() {
	sshKeyCmd.AddCommand(sshKeyAddCmd)
	sshKeyAddCmd.Flags().StringP("name", "n", "", "the name of the key")
	sshKeyAddCmd.Flags().BoolP("generate", "g", false, "generates the key if not exist")
	sshKeyAddCmd.Flags().String("private-key-path", "~/.ssh/id_rsa", "the path to the private key")
	sshKeyAddCmd.Flags().String("public-key-path", "~/.ssh/id_rsa.pub", "the path to the public key")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// sshKeyAddCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// sshKeyAddCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
