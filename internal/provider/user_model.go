package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

// UserModel maps InfluxDB User schema data.
type UserModel struct {
	Id       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Password types.String `tfsdk:"password"`
	OrgId    types.String `tfsdk:"org_id"`
	OrgRole  types.String `tfsdk:"org_role"`
	Status   types.String `tfsdk:"status"`
}

// getUserOrgMembership gets the organization membership information for a user.
func getUserOrgMembership(ctx context.Context, client influxdb2.Client, userID string) (orgID string, orgRole string, err error) {
	// Get all organizations
	orgs, err := client.OrganizationsAPI().GetOrganizations(ctx)
	if err != nil {
		return "", "", fmt.Errorf("failed to get organizations: %w", err)
	}

	// Check each organization for user membership
	for _, org := range *orgs {
		if org.Id == nil {
			continue
		}

		// Check if user is an owner
		owners, err := client.OrganizationsAPI().GetOwnersWithID(ctx, *org.Id)
		if err == nil && owners != nil {
			for _, owner := range *owners {
				if owner.Id != nil && *owner.Id == userID {
					return *org.Id, "owner", nil
				}
			}
		}

		// Check if user is a member
		members, err := client.OrganizationsAPI().GetMembersWithID(ctx, *org.Id)
		if err == nil && members != nil {
			for _, member := range *members {
				if member.Id != nil && *member.Id == userID {
					return *org.Id, "member", nil
				}
			}
		}
	}

	// User is not a member of any organization
	return "", "", nil
}
