/*
Copyright 2021 The cert-manager Authors.

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

package options

import (
	"flag"
	"fmt"

	cmclient "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/klog/v2"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

const defaultPrometheusMetricsServerAddress = "0.0.0.0:9402"

// Options are the main options for the driver. Populated via processing
// command line flags.
type Options struct {
	// logLevel is the verbosity level the driver will write logs at.
	logLevel string

	// kubeConfigFlags handles the Kubernetes authentication flags and builds a useable rest config.
	kubeConfigFlags *genericclioptions.ConfigFlags

	// NodeID is the name of the node which is hosting this driver instance.
	NodeID string

	// DriverName is the name of this CSI driver which will be shared with
	// the Kubelet.
	DriverName string

	// Endpoint is the endpoint that the driver will connect to the Kubelet.
	Endpoint string

	// DataRoot is the directory that the driver will write and mount volumes
	// from.
	DataRoot string

	// UseTokenRequest declares that the CSI driver will use the empty audience
	// token request for creating CertificateRequests. Requires the token request
	// to be defined on the CSIDriver manifest.
	UseTokenRequest bool

	// Logr is the shared base logger.
	Logr logr.Logger

	// RestConfig is the shared base rest config to connect to the Kubernetes
	// API.
	RestConfig *rest.Config

	// CMClient is a rest client for interacting with cert-manager resources.
	CMClient cmclient.Interface

	// The host and port that the metrics endpoint should listen on.
	MetricsListenAddress string
}

func New() *Options {
	return new(Options)
}

func (o *Options) Prepare(cmd *cobra.Command) *Options {
	o.addFlags(cmd)
	return o
}

func (o *Options) Complete() error {
	klog.InitFlags(nil)
	log := klog.TODO()
	if err := flag.Set("v", o.logLevel); err != nil {
		return fmt.Errorf("failed to set log level: %s", err)
	}
	o.Logr = log

	var err error
	o.RestConfig, err = o.kubeConfigFlags.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("failed to build kubernetes rest config: %s", err)
	}

	o.CMClient, err = cmclient.NewForConfig(o.RestConfig)
	if err != nil {
		return fmt.Errorf("failed to build cert-manager rest client: %s", err)
	}

	return nil
}

func (o *Options) addFlags(cmd *cobra.Command) {
	var nfs cliflag.NamedFlagSets

	o.addAppFlags(nfs.FlagSet("App"))
	o.kubeConfigFlags = genericclioptions.NewConfigFlags(true)
	o.kubeConfigFlags.AddFlags(nfs.FlagSet("Kubernetes"))

	usageFmt := "Usage:\n  %s\n"
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStderr(), nfs, 0)
		return nil
	})

	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStdout(), nfs, 0)
	})

	fs := cmd.Flags()
	for _, f := range nfs.FlagSets {
		fs.AddFlagSet(f)
	}
}

func (o *Options) addAppFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.logLevel,
		"log-level", "v", "1",
		"Log level (1-5).")

	fs.StringVar(&o.NodeID, "node-id", "",
		"The name of the node which is hosting this driver instance.")
	if err := cobra.MarkFlagRequired(fs, "node-id"); err != nil {
		panic(err)
	}

	fs.StringVar(&o.Endpoint, "endpoint", "",
		"The endpoint that the driver will connect to the Kubelet.")
	if err := cobra.MarkFlagRequired(fs, "endpoint"); err != nil {
		panic(err)
	}

	fs.StringVar(&o.DriverName, "driver-name", "csi.cert-manager.io",
		"The name of this CSI driver which will be shared with the Kubelet.")

	fs.StringVar(&o.DataRoot, "data-root", "/csi-data-dir",
		"The directory that the driver will write and mount volumes from.")

	fs.BoolVar(&o.UseTokenRequest, "use-token-request", false,
		"Use the empty audience token request for creating CertificateRequests. Requires the token request to be defined on the CSIDriver manifest.")
	fs.StringVar(&o.MetricsListenAddress, "metrics-listen-address", defaultPrometheusMetricsServerAddress,
		"The host and port that the metrics endpoint should listen on.")
}
