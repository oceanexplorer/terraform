package jsonplan

import "encoding/json"

// Resource is the representation of a resource in the json plan
type resource struct {
	// Address is the absolute resource address
	Address string `json:"address"`

	// Mode can be "managed" or "data"
	Mode string `json:"mode"`

	Type string `json:"type"`
	Name string `json:"name"`

	// Index is omitted for a resource not using `count` or `for_each`
	Index int `json:"index,omitempty"`

	// ProviderName allows the property "type" to be interpreted unambiguously
	// in the unusual situation where a provider offers a resource type whose
	// name does not start with its own name, such as the "googlebeta" provider
	// offering "google_compute_instance".
	ProviderName string `json:"provider_name"`

	// SchemaVersion indicates which version of the resource type schema the
	// "values" property conforms to.
	SchemaVersion int `json:"schema_version"`

	// Values is the JSON representation of the attribute values of the
	// resource, whose structure depends on the resource type schema. Any
	// unknown values are omitted or set to null, making them indistinguishable
	// from absent values.
	Values json.RawMessage `json:"values"`
}

// resourceChange is a description of an individual change action that Terraform
// plans to use to move from the prior state to a new state matching the
// configuration.

type resourceChange struct {
	// Address is the absolute resource address
	Address string `json:"address,omitempty"`

	// ModuleAddress is the module portion of the above address. Omitted if the
	// instance is in the root module.
	ModuleAddress string `json:"module_address,omitempty"`

	// "managed" or "data"
	Mode string

	Type  string
	Name  string
	Index string

	// "deposed", if set, indicates that this action applies to a "deposed"
	// object of the given instance rather than to its "current" object. Omitted
	// for changes to the current object.
	Deposed bool `json:"deposed,omitempty"`

	// Change describes the change that will be made to this object
	Change change
}
