// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package pulsar

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/streamnative/pulsarctl/pkg/pulsar"
	"github.com/streamnative/pulsarctl/pkg/pulsar/common"
)

// Provider returns a terraform.ResourceProvider
func Provider() *schema.Provider {
	provider := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"web_service_url": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions["web_service_url"],
				DefaultFunc: schema.EnvDefaultFunc("WEB_SERVICE_URL", nil),
			},
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PULSAR_AUTH_TOKEN", nil),
				Description: descriptions["token"],
			},
			"api_version": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "1",
				Deprecated:  "The newer versions can use the right version for the right type of resource",
				Description: descriptions["api_version"],
			},
			"tls_trust_certs_file_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["tls_trust_certs_file_path"],
				DefaultFunc: schema.EnvDefaultFunc("TLS_TRUST_CERTS_FILE_PATH", nil),
			},
			"tls_allow_insecure_connection": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: descriptions["tls_allow_insecure_connection"],
				DefaultFunc: schema.EnvDefaultFunc("TLS_ALLOW_INSECURE_CONNECTION", false),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"pulsar_tenant":    resourcePulsarTenant(),
			"pulsar_cluster":   resourcePulsarCluster(),
			"pulsar_namespace": resourcePulsarNamespace(),
			"pulsar_topic":     resourcePulsarTopic(),
			"pulsar_sink":      resourcePulsarSink(),
		},
	}

	provider.ConfigureContextFunc = func(_ context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		tfVersion := provider.TerraformVersion
		if tfVersion == "" {
			// Terraform 0.12 introduced this field to the protocol, so if this field is missing,
			// we can assume Terraform version is <= 0.11
			tfVersion = "0.11+compatible"
		}

		if err := validatePulsarConfig(d); err != nil {
			return nil, diag.FromErr(err)
		}

		output, err := providerConfigure(d, tfVersion)
		if err != nil {
			return nil, diag.FromErr(err)
		}
		return output, nil
	}

	return provider
}

func providerConfigure(d *schema.ResourceData, tfVersion string) (interface{}, error) {
	// can be used for version locking or version specific feature sets
	log.Printf("Started")
	_ = tfVersion
	clusterURL := d.Get("web_service_url").(string)
	token := d.Get("token").(string)
	TLSTrustCertsFilePath := d.Get("tls_trust_certs_file_path").(string)
	TLSAllowInsecureConnection := d.Get("tls_allow_insecure_connection").(bool)

	meta := make(map[common.APIVersion]pulsar.Client, 3)
	for _, version := range []common.APIVersion{common.V1, common.V2, common.V3} {
		config := &common.Config{
			WebServiceURL:              clusterURL,
			Token:                      token,
			PulsarAPIVersion:           version,
			TLSTrustCertsFilePath:      TLSTrustCertsFilePath,
			TLSAllowInsecureConnection: TLSAllowInsecureConnection,
		}

		client, err := pulsar.New(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create pulsar client: %w", err)
		}
		meta[version] = client
	}
	return meta, nil
}

func validatePulsarConfig(d *schema.ResourceData) error {
	webServiceURL := d.Get("web_service_url").(string)

	if _, err := url.Parse(webServiceURL); err != nil {
		return fmt.Errorf("ERROR_PULSAR_CONFIG_INVALID_WEB_SERVICE_URL: %w", err)
	}

	return nil
}

var descriptions map[string]string

func init() {
	descriptions = map[string]string{
		"web_service_url": "Web service url is used to connect to your apache pulsar cluster",
		"token": `Authentication Token used to grant terraform permissions
to modify Apace Pulsar Entities`,
		"api_version":                   "Api Version to be used for the pulsar admin interaction",
		"tls_trust_certs_file_path":     "Path to a custom trusted TLS certificate file",
		"tls_allow_insecure_connection": "Boolean flag to accept untrusted TLS certificates",
		"admin_roles":                   "Admin roles to be attached to tenant",
		"allowed_clusters":              "Tenant will be able to interact with these clusters",
		"namespace":                     "Pulsar namespaces are logical groupings of topics",
		"tenant": `An administrative unit for allocating capacity and enforcing an 
authentication/authorization scheme`,
		"namespace_list": "List of namespaces for a given tenant",
		"enable_duplication": `ensures that each message produced on Pulsar topics is persisted to disk 
only once, even if the message is produced more than once`,
		"encrypt_topics":                 "encrypt messages at the producer and decrypt at the consumer",
		"max_producers_per_topic":        "Max number of producers per topic",
		"max_consumers_per_subscription": "Max number of consumers per subscription",
		"max_consumers_per_topic":        "Max number of consumers per topic",
		"dispatch_rate":                  "Data transfer rate, in and out of the Pulsar Broker",
		"persistence_policy":             "Policy for the namespace for data persistence",
		"backlog_quota":                  "",
		resourceConfigsAttribute:         "Configuration encoded as JSON",
	}
}
