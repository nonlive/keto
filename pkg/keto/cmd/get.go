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
	"os"

	"github.com/UKHomeOffice/keto/pkg/keto"

	"github.com/spf13/cobra"
)

// getCmd represents the 'get' command
var getCmd = &cobra.Command{
	Use:        "get <subcommand>",
	Short:      "Get resources",
	SuggestFor: []string{"list"},
}

var getClusterCmd = &cobra.Command{
	Use:          "cluster [NAME]",
	Aliases:      clusterCmdAliases,
	Short:        "Get clusters",
	Long:         "Get clusters",
	SuggestFor:   []string{"clusters"},
	SilenceUsage: true,
	RunE: func(c *cobra.Command, args []string) error {
		return getClusterCmdFunc(c, args)
	},
}

func getClusterCmdFunc(c *cobra.Command, args []string) error {
	name := ""
	if len(args) == 1 {
		name = args[0]
	}

	cli, err := newCLI(c)
	if err != nil {
		return err
	}

	return listClusters(cli, name)
}

var getMasterPoolCmd = &cobra.Command{
	Use:          "masterpool [NAME]",
	Aliases:      masterPoolCmdAliases,
	Short:        "Get master pools",
	Long:         "Get master pools",
	SuggestFor:   []string{"masters", "pool"},
	SilenceUsage: true,
	RunE: func(c *cobra.Command, args []string) error {
		return getMasterPoolCmdFunc(c, args)
	},
}

func getMasterPoolCmdFunc(c *cobra.Command, args []string) error {
	clusterName, err := c.Flags().GetString("cluster")
	if err != nil {
		return err
	}

	name := ""
	if len(args) == 1 {
		name = args[0]
	}

	cli, err := newCLI(c)
	if err != nil {
		return err
	}

	return listMasterPools(cli, clusterName, name)
}

var getComputePoolCmd = &cobra.Command{
	Use:          "computepool [NAME]",
	Aliases:      computePoolCmdAliases,
	Short:        "Get compute pools",
	Long:         "Get compute pools",
	SuggestFor:   []string{"compute", "pool"},
	SilenceUsage: true,
	RunE: func(c *cobra.Command, args []string) error {
		return getComputePoolCmdFunc(c, args)
	},
}

func getComputePoolCmdFunc(c *cobra.Command, args []string) error {
	clusterName, err := c.Flags().GetString("cluster")
	if err != nil {
		return err
	}

	name := ""
	if len(args) == 1 {
		name = args[0]
	}

	cli, err := newCLI(c)
	if err != nil {
		return err
	}

	return listComputePools(cli, clusterName, name)
}

func listMasterPools(cli *cli, clusterName, name string) error {
	pools, err := cli.ctrl.GetMasterPools(clusterName, name)
	if err != nil {
		return err
	}
	return keto.PrintMasterPool(keto.GetPrinter(os.Stdout), pools, true)
}

func listComputePools(cli *cli, clusterName, name string) error {
	pools, err := cli.ctrl.GetComputePools(clusterName, name)
	if err != nil {
		return err
	}
	return keto.PrintComputePool(keto.GetPrinter(os.Stdout), pools, true)
}

func listClusters(cli *cli, name string) error {
	clusters, err := cli.ctrl.GetClusters(name)
	if err != nil {
		return err
	}
	return keto.PrintClusters(keto.GetPrinter(os.Stdout), clusters, true)
}

func init() {
	getCmd.AddCommand(
		getClusterCmd,
		getMasterPoolCmd,
		getComputePoolCmd,
	)

	// Add flags that are relevant to different subcommands.
	addClusterFlag(
		getMasterPoolCmd,
		getComputePoolCmd,
	)
}
