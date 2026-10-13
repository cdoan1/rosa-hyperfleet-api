package render

import (
	"bytes"
	"encoding/json"

	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
)

// MarshalApplyObject serializes a rendered resource for server-side apply.
// OpenShift's OIDCProvider Go types serialize the optional oidcClients and
// username.prefix fields as null when they are nil. HostedCluster rejects null
// for these fields, so leave them out of the apply payload instead.
func MarshalApplyObject(object any) ([]byte, error) {
	content, err := json.Marshal(object)
	if err != nil {
		return nil, err
	}
	if _, ok := object.(*hypershiftv1beta1.HostedCluster); !ok {
		return content, nil
	}

	return omitNullOIDCProviderFields(content)
}

func omitNullOIDCProviderFields(content []byte) ([]byte, error) {
	root, err := unmarshalJSONObject(content)
	if err != nil {
		return nil, err
	}
	if root == nil {
		return content, nil
	}

	spec, err := unmarshalJSONObject(root["spec"])
	if err != nil || spec == nil {
		return content, err
	}
	configuration, err := unmarshalJSONObject(spec["configuration"])
	if err != nil || configuration == nil {
		return content, err
	}
	authentication, err := unmarshalJSONObject(configuration["authentication"])
	if err != nil || authentication == nil {
		return content, err
	}

	providersJSON, ok := authentication["oidcProviders"]
	if !ok {
		return content, nil
	}
	var providers []json.RawMessage
	if err := json.Unmarshal(providersJSON, &providers); err != nil {
		return nil, err
	}
	if providers == nil {
		return content, nil
	}

	changed := false
	for i, providerJSON := range providers {
		provider, err := unmarshalJSONObject(providerJSON)
		if err != nil {
			return nil, err
		}
		if provider == nil {
			continue
		}

		providerChanged := removeNullField(provider, "oidcClients")
		claimMappings, err := unmarshalJSONObject(provider["claimMappings"])
		if err != nil {
			return nil, err
		}
		if claimMappings != nil {
			username, err := unmarshalJSONObject(claimMappings["username"])
			if err != nil {
				return nil, err
			}
			if username != nil && removeNullField(username, "prefix") {
				claimMappings["username"], err = json.Marshal(username)
				if err != nil {
					return nil, err
				}
				provider["claimMappings"], err = json.Marshal(claimMappings)
				if err != nil {
					return nil, err
				}
				providerChanged = true
			}
		}
		if !providerChanged {
			continue
		}

		providers[i], err = json.Marshal(provider)
		if err != nil {
			return nil, err
		}
		changed = true
	}
	if !changed {
		return content, nil
	}

	authentication["oidcProviders"], err = json.Marshal(providers)
	if err != nil {
		return nil, err
	}
	configuration["authentication"], err = json.Marshal(authentication)
	if err != nil {
		return nil, err
	}
	spec["configuration"], err = json.Marshal(configuration)
	if err != nil {
		return nil, err
	}
	root["spec"], err = json.Marshal(spec)
	if err != nil {
		return nil, err
	}

	return json.Marshal(root)
}

func unmarshalJSONObject(data json.RawMessage) (map[string]json.RawMessage, error) {
	if len(data) == 0 || bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil, nil
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		return nil, err
	}
	return object, nil
}

func removeNullField(object map[string]json.RawMessage, field string) bool {
	value, ok := object[field]
	if !ok || !bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		return false
	}
	delete(object, field)
	return true
}
