package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config/configschema"
	"github.com/hashicorp/terraform/providers"
	"github.com/zclconf/go-cty/cty"
)

// Provider is an implementation of providers.Interface
type Provider struct {
	// Provider is the schema for the provider itself.
	Schema providers.Schema

	// DataSources maps the data source name to that data source's schema.
	DataSources map[string]providers.Schema
}

// NewProvider returns a new terraform provider
// of some sort.
// Not sure what of this belongs in Configure()
func NewProvider() *Provider {
	return &Provider{
		Schema: providers.Schema{
			Version: 1,
		},
		DataSources: map[string]providers.Schema{
			"terraform_remote_state": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"backend": {
							Type:     cty.String,
							Required: true,
						},
						"config": {
							Type:     cty.DynamicPseudoType,
							Optional: true,
						},
						"defaults": {
							Type:     cty.DynamicPseudoType,
							Optional: true,
						},
						"outputs": {
							Type:     cty.DynamicPseudoType,
							Computed: true,
						},
						"workspace": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

// GetSchema returns the complete schema for the provider.
func (t *Provider) GetSchema() providers.GetSchemaResponse {
	return providers.GetSchemaResponse{
		Provider:    t.Schema,
		DataSources: t.DataSources,
		// Diagnostics ?
	}
}

// ValidateProviderConfig is used to validate the configuration values.
func (t *Provider) ValidateProviderConfig(providers.ValidateProviderConfigRequest) providers.ValidateProviderConfigResponse {
	// At this moment there is nothing to configure for the terraform provider,
	// so we will happily return without taking any action
	var res providers.ValidateProviderConfigResponse
	return res
}

// ValidateDataSourceConfig is used to validate the data source configuration values.
func (t *Provider) ValidateDataSourceConfig(providers.ValidateDataSourceConfigRequest) providers.ValidateDataSourceConfigResponse {
	// At this moment there is nothing to configure for the terraform provider,
	// so we will happily return without taking any action
	var res providers.ValidateDataSourceConfigResponse
	return res
}

// Configure configures and initializes the provider.
func (t *Provider) Configure(providers.ConfigureRequest) providers.ConfigureResponse {
	// At this moment there is nothing to configure for the terraform provider,
	// so we will happily return without taking any action
	var res providers.ConfigureResponse
	return res
}

// ReadDataSource returns the data source's current state.
func (t *Provider) ReadDataSource(req providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
	// call function
	var res providers.ReadDataSourceResponse

	// This should not happen
	if req.TypeName != "terraform_remote_state" {
		res.Diagnostics.Append(fmt.Errorf("Error: unsupported data source %s", req.TypeName))
		return res
	}

	// something something call this function dataSourceRemoteStateRead()

	return res
}

// Stop is called when the provider should halt any in-flight actions.
func (t *Provider) Stop() error {
	panic("unimplemented")
}

// All the Resource-specific functions are below.
// The terraform provider supplies a single data source, `terraform_remote_state`
// and no resources.

// UpgradeResourceState is called when the state loader encounters an
// instance state whose schema version is less than the one reported by the
// currently-used version of the corresponding provider, and the upgraded
// result is used for any further processing.
func (t *Provider) UpgradeResourceState(providers.UpgradeResourceStateRequest) providers.UpgradeResourceStateResponse {
	panic("unimplemented - terraform_remote_state has no resources")
}

// ReadResource refreshes a resource and returns its current state.
func (t *Provider) ReadResource(providers.ReadResourceRequest) providers.ReadResourceResponse {
	panic("unimplemented - terraform_remote_state has no resources")
}

// PlanResourceChange takes the current state and proposed state of a
// resource, and returns the planned final state.
func (t *Provider) PlanResourceChange(providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
	panic("unimplemented - terraform_remote_state has no resources")
}

// ApplyResourceChange takes the planned state for a resource, which may
// yet contain unknown computed values, and applies the changes returning
// the final state.
func (t *Provider) ApplyResourceChange(providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
	panic("unimplemented - terraform_remote_state has no resources")
}

// ImportResourceState requests that the given resource be imported.
func (t *Provider) ImportResourceState(providers.ImportResourceStateRequest) providers.ImportResourceStateResponse {
	panic("unimplemented - terraform_remote_state has no resources")
}

// ValidateResourceTypeConfig is used to to validate the resource configuration values.
func (t *Provider) ValidateResourceTypeConfig(providers.ValidateResourceTypeConfigRequest) providers.ValidateResourceTypeConfigResponse {
	// At this moment there is nothing to configure for the terraform provider,
	// so we will happily return without taking any action
	var res providers.ValidateResourceTypeConfigResponse
	return res
}
