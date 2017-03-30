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
	"os"
	"strings"

	"github.com/UKHomeOffice/keto/pkg/keto"
	cmdutil "github.com/UKHomeOffice/keto/pkg/keto/cmd/util"

	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:          "get <" + strings.Join(resourceTypes, "|") + "> [NAME]",
	Short:        "Get a resource",
	Long:         "Get a resource",
	SuggestFor:   []string{"list"},
	ValidArgs:    resourceTypes,
	SilenceUsage: true,
	PreRunE: func(c *cobra.Command, args []string) error {
		return validateGetFlags(c, args)
	},
	RunE: func(c *cobra.Command, args []string) error {
		return runGet(c, args)
	},
}

func validateGetFlags(c *cobra.Command, args []string) error {
	validTypes := "Valid types: " + strings.Join(resourceTypes, ", ")

	if len(args) < 1 {
		return fmt.Errorf("resource type not specified. " + validTypes)
	}

	if !cmdutil.StringInSlice(args[0], resourceTypes) {
		return fmt.Errorf("invalid resource type. " + validTypes)
	}
	return nil
}

func runGet(c *cobra.Command, args []string) error {
	client, err := newClient(c)
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
		if err := listNodePools(client, clusterName, resName); err != nil {
			return err
		}
	} else {
		// TODO: implement listing clusters
		return errors.New("not implemented")
	}
	return nil
}

func listNodePools(client *client, clusterName, poolName string) error {
	pools, err := client.ctrl.ListNodePools(clusterName)
	if err != nil {
		return err
	}
	if err := keto.PrintNodePool(keto.GetPrinter(os.Stdout), pools, true); err != nil {
		return err
	}
	return nil
}

func init() {
	RootCmd.AddCommand(getCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// getCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// getCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	addClusterFlag(getCmd)
}
