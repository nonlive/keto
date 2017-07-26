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
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"

	"github.com/UKHomeOffice/keto/pkg/constants"
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

	cli, err := newCLI(c)
	if err != nil {
		return err
	}

	return listClusters(cli, args...)
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

	cli, err := newCLI(c)
	if err != nil {
		return err
	}

	return listMasterPools(cli, clusterName, args...)
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

	cli, err := newCLI(c)
	if err != nil {
		return err
	}

	return listComputePools(cli, clusterName, args...)
}

func listMasterPools(cli *cli, clusterName string, names ...string) error {
	pools, err := cli.ctrl.GetMasterPools(clusterName, names...)
	if err != nil {
		return err
	}
	return keto.PrintMasterPool(keto.GetPrinter(os.Stdout), pools, true)
}

func listComputePools(cli *cli, clusterName string, names ...string) error {
	pools, err := cli.ctrl.GetComputePools(clusterName, names...)
	if err != nil {
		return err
	}
	return keto.PrintComputePool(keto.GetPrinter(os.Stdout), pools, true)
}

func listClusters(cli *cli, names ...string) error {
	clusters, err := cli.ctrl.GetClusters(names...)
	if err != nil {
		return err
	}
	return keto.PrintClusters(keto.GetPrinter(os.Stdout), clusters, true)
}

var getClusterConfigCmd = &cobra.Command{
	Use:          "config --cluster [NAME]",
	Aliases:      clusterConfigCmdAliases,
	Short:        "Get cluster kubernetes config",
	Long:         "Get cluster kubernetes config",
	SilenceUsage: true,
	PreRunE: func(c *cobra.Command, args []string) error {
		return validateGetConfigPrecursors(c, args)
	},
	RunE: func(c *cobra.Command, args []string) error {
		return getClusterConfigCmdFunc(c, args)
	},
}

func validateGetConfigPrecursors(c *cobra.Command, args []string) error {
	if !c.Flags().Changed("cluster") {
		return fmt.Errorf("cluster name must be set: --cluster [NAME]")
	}

	_, err := exec.LookPath("kubeadm")
	if err != nil {
		var binaryDownloadLink bytes.Buffer
		binaryDownloadLink.WriteString("https://storage.googleapis.com/kubernetes-release/release/")
		binaryDownloadLink.WriteString(constants.DefaultKubeVersion)
		binaryDownloadLink.WriteString("/bin/")
		binaryDownloadLink.WriteString(runtime.GOOS + "/")
		binaryDownloadLink.WriteString(runtime.GOARCH)
		binaryDownloadLink.WriteString("/kubeadm")

		return fmt.Errorf(`the executable 'kubeadm' was not found. Retrieve it using the following command:
curl -Lo %q /usr/local/bin/kubeadm && chmod +x /usr/local/bin/kubeadm
Note: If the 'kubeadm' binary is not built for your distribution the above link may not work.`, binaryDownloadLink.String())
	}

	_, err = c.Flags().GetString("cluster")
	if err != nil {
		return err
	}

	assetsDir, err := c.Flags().GetString("assets-dir")
	if err != nil {
		return err
	}
	if assetsDir == "" {
		assetsDir, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	// Check if assets exist, create symlinks from 'kube_ca' files if necessary
	certName := "ca.crt"
	keyName := "ca.key"

	if !fileExists(assetsDir) {
		return fmt.Errorf("assets directory %q does not exist", assetsDir)
	}
	if !fileExists(path.Join(assetsDir, certName)) {
		if !fileExists(path.Join(assetsDir, "kube_" + certName)) {
			return fmt.Errorf("%q does not exist (kube ca certificate)", path.Join(assetsDir, certName))
		}
		os.Symlink(path.Join(assetsDir, "kube_" + certName), path.Join(assetsDir, certName))
	}
	if !fileExists(path.Join(assetsDir, keyName)) {
		if !fileExists(path.Join(assetsDir, "kube_" + keyName)) {
			return fmt.Errorf("%q does not exist (kube ca private key)", path.Join(assetsDir, keyName))
		}
		os.Symlink(path.Join(assetsDir, "kube_" + keyName), path.Join(assetsDir, keyName))
	}

	return nil
}

func getClusterConfigCmdFunc(c *cobra.Command, args []string) error {
	clusterName, _ := c.Flags().GetString("cluster")
	assetsDir, _ := c.Flags().GetString("assets-dir")
	outputFile, _ := c.Flags().GetString("output-file")

	cli, err := newCLI(c)
	if err != nil {
		return err
	}

	config, err := cli.ctrl.GetClusterConfig(clusterName, assetsDir)
	if err != nil {
		return err
	}

	return keto.PrintClusterConfig(keto.GetPrinter(os.Stdout), config, outputFile)
}

func init() {
	getCmd.AddCommand(
		getClusterCmd,
		getMasterPoolCmd,
		getComputePoolCmd,
		getClusterConfigCmd,
	)

	// Add flags that are relevant to different subcommands.
	addClusterFlag(
		getMasterPoolCmd,
		getComputePoolCmd,
		getClusterConfigCmd,
	)

	addAssetsDirFlag(
		getClusterConfigCmd,
	)

	addConfigOutputFileFlag(
		getClusterConfigCmd,
	)
}
