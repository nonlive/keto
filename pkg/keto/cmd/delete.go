/*
Copyright 2017 The Keto Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/UKHomeOffice/keto/pkg/constants"
	cmdutil "github.com/UKHomeOffice/keto/pkg/keto/cmd/util"

	"github.com/spf13/cobra"
)

// deleteCmd represents the create command
var deleteCmd = &cobra.Command{
	Use:          "delete <" + strings.Join(constants.ValidResourceTypes, "|") + "> <NAME>",
	Short:        "Delete a resource",
	Long:         "Delete a resource",
	SuggestFor:   []string{"remove"},
	ValidArgs:    constants.ValidResourceTypes,
	SilenceUsage: true,
	PreRunE: func(c *cobra.Command, args []string) error {
		return validateDeleteFlags(c, args)
	},
	RunE: func(c *cobra.Command, args []string) error {
		return runDelete(c, args)
	},
}

func validateDeleteFlags(c *cobra.Command, args []string) error {
	validTypes := "Valid types: " + strings.Join(constants.ValidResourceTypes, ", ")

	if len(args) < 1 {
		return fmt.Errorf("resource type not specified. " + validTypes)
	}

	if !cmdutil.StringInSlice(args[0], constants.ValidResourceTypes) {
		return fmt.Errorf("invalid resource type. " + validTypes)
	}
	return nil
}

func runDelete(c *cobra.Command, args []string) error {
	cli, err := newCLI(c)
	if err != nil {
		return err
	}

	clusterName, err := c.Flags().GetString("cluster")
	if err != nil {
		return err
	}
	res := args[0]
	resName := ""
	if len(args) == 2 {
		resName = args[1]
	}

	if res == "nodepool" {
		if err := deleteNodePool(cli, clusterName, resName); err != nil {
			return err
		}
	} else {
		// TODO: implement deleting clusters
		return errors.New("not implemented")
	}
	return nil
}

func deleteNodePool(cli *cli, clusterName, name string) error {
	return cli.ctrl.DeleteNodePool(clusterName, name)
}

func init() {
	RootCmd.AddCommand(deleteCmd)

	// Add flags that are relevant to delete cmd.
	addClusterFlag(deleteCmd)
}
