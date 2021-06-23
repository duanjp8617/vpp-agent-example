//  Copyright (c) 2019 Cisco and/or its affiliates.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at:
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package main

import (
	"fmt"
	"log"

	"go.ligato.io/cn-infra/v2/agent"
	"go.ligato.io/cn-infra/v2/datasync"
	"go.ligato.io/cn-infra/v2/datasync/kvdbsync"
	"go.ligato.io/cn-infra/v2/datasync/kvdbsync/local"
	"go.ligato.io/cn-infra/v2/datasync/resync"
	"go.ligato.io/cn-infra/v2/db/keyval/etcd"
	"go.ligato.io/cn-infra/v2/health/statuscheck"

	"go.ligato.io/vpp-agent/v3/client"
	legacyclient "go.ligato.io/vpp-agent/v3/clientv2/linux/localclient"
	"go.ligato.io/vpp-agent/v3/plugins/orchestrator"
	vpp_ifplugin "go.ligato.io/vpp-agent/v3/plugins/vpp/ifplugin"
	vpp_l2plugin "go.ligato.io/vpp-agent/v3/plugins/vpp/l2plugin"
	vpp_l2 "go.ligato.io/vpp-agent/v3/proto/ligato/vpp/l2"
)

func main() {
	ep := &ExamplePlugin{
		VPPIfPlugin:  &vpp_ifplugin.DefaultPlugin,
		VPPL2Plugin:  &vpp_l2plugin.DefaultPlugin,
		Orchestrator: &orchestrator.DefaultPlugin,
		ETCDDataSync: kvdbsync.NewPlugin(kvdbsync.UseKV(&etcd.DefaultPlugin)),
	}

	writers := datasync.KVProtoWriters{
		ep.ETCDDataSync,
	}
	statuscheck.DefaultPlugin.Transport = writers

	watchers := datasync.KVProtoWatchers{
		local.DefaultRegistry,
		ep.ETCDDataSync,
	}
	orchestrator.DefaultPlugin.Watcher = watchers

	a := agent.NewAgent(
		agent.AllPlugins(ep),
	)
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}

// ExamplePlugin is the main plugin which
// handles resync and changes in this example.
type ExamplePlugin struct {
	VPPIfPlugin  *vpp_ifplugin.IfPlugin
	VPPL2Plugin  *vpp_l2plugin.L2Plugin
	Orchestrator *orchestrator.Plugin
	ETCDDataSync *kvdbsync.Plugin
}

// String returns plugin name
func (p *ExamplePlugin) String() string {
	return "example"
}

// Init handles initialization phase.
func (p *ExamplePlugin) Init() error {
	return nil
}

// AfterInit first triggers localclient-based resync, then resync against etcd.
func (p *ExamplePlugin) AfterInit() error {
	// local client resync
	resyncLocalClient()

	// demonstrate also legacy localclient - it should not trigger any additional
	// changes since the same configuration was already applied by resyncLocalClient().
	resyncLegacyLocalClient()

	// etcd resync
	fmt.Println("=== ETCD RESYNC ===")
	resync.DefaultPlugin.DoResync()
	return nil
}

// Close cleans up the resources.
func (p *ExamplePlugin) Close() error {
	return nil
}

// resyncLegacyLocalClient demonstrates resync of the local client (from "client"
// package)
func resyncLocalClient() {
	fmt.Println("=== LOCALCLIENT RESYNC ===")

	err := client.LocalClient.ResyncConfig(bd1)
	if err != nil {
		fmt.Println(err)
		return
	}
}

// resyncLegacyLocalClient demonstrates resync of (legacy) local client
// ("clientv2" package). It is recommended to use the client from "client"
// package instead, simply because it is extensible beyond the vpp-agent core
// plugins and also it provides additional methods to obtain the configuration
// state.
func resyncLegacyLocalClient() {
	fmt.Println("=== LEGACY LOCALCLIENT RESYNC ===")

	txn := legacyclient.DataResyncRequest("example")
	err := txn.BD(bd1).
		Send().ReceiveReply()
	if err != nil {
		fmt.Println(err)
		return
	}
}

var (
	bd1 = &vpp_l2.BridgeDomain{
		Name: "bd1",
		Interfaces: []*vpp_l2.BridgeDomain_Interface{
			{Name: "tap0"},
			{Name: "tap1"},
		},
	}
)
