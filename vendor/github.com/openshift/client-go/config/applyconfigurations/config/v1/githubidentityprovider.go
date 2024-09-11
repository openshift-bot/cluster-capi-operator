// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

// GitHubIdentityProviderApplyConfiguration represents an declarative configuration of the GitHubIdentityProvider type for use
// with apply.
type GitHubIdentityProviderApplyConfiguration struct {
	ClientID      *string                                   `json:"clientID,omitempty"`
	ClientSecret  *SecretNameReferenceApplyConfiguration    `json:"clientSecret,omitempty"`
	Organizations []string                                  `json:"organizations,omitempty"`
	Teams         []string                                  `json:"teams,omitempty"`
	Hostname      *string                                   `json:"hostname,omitempty"`
	CA            *ConfigMapNameReferenceApplyConfiguration `json:"ca,omitempty"`
}

// GitHubIdentityProviderApplyConfiguration constructs an declarative configuration of the GitHubIdentityProvider type for use with
// apply.
func GitHubIdentityProvider() *GitHubIdentityProviderApplyConfiguration {
	return &GitHubIdentityProviderApplyConfiguration{}
}

// WithClientID sets the ClientID field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ClientID field is set to the value of the last call.
func (b *GitHubIdentityProviderApplyConfiguration) WithClientID(value string) *GitHubIdentityProviderApplyConfiguration {
	b.ClientID = &value
	return b
}

// WithClientSecret sets the ClientSecret field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ClientSecret field is set to the value of the last call.
func (b *GitHubIdentityProviderApplyConfiguration) WithClientSecret(value *SecretNameReferenceApplyConfiguration) *GitHubIdentityProviderApplyConfiguration {
	b.ClientSecret = value
	return b
}

// WithOrganizations adds the given value to the Organizations field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Organizations field.
func (b *GitHubIdentityProviderApplyConfiguration) WithOrganizations(values ...string) *GitHubIdentityProviderApplyConfiguration {
	for i := range values {
		b.Organizations = append(b.Organizations, values[i])
	}
	return b
}

// WithTeams adds the given value to the Teams field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Teams field.
func (b *GitHubIdentityProviderApplyConfiguration) WithTeams(values ...string) *GitHubIdentityProviderApplyConfiguration {
	for i := range values {
		b.Teams = append(b.Teams, values[i])
	}
	return b
}

// WithHostname sets the Hostname field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Hostname field is set to the value of the last call.
func (b *GitHubIdentityProviderApplyConfiguration) WithHostname(value string) *GitHubIdentityProviderApplyConfiguration {
	b.Hostname = &value
	return b
}

// WithCA sets the CA field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the CA field is set to the value of the last call.
func (b *GitHubIdentityProviderApplyConfiguration) WithCA(value *ConfigMapNameReferenceApplyConfiguration) *GitHubIdentityProviderApplyConfiguration {
	b.CA = value
	return b
}
