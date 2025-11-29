package organizations

// Schema names for OpenAPI documentation.
// These constants ensure type-safe schema references in route metadata.
const (
	// Request schemas
	SchemaCreateOrganizationRequest    = "CreateOrganizationRequest"
	SchemaUpdateOrganizationRequest    = "UpdateOrganizationRequest"
	SchemaAddOrganizationMemberRequest = "AddOrganizationMemberRequest"
	SchemaUpdateMemberRoleRequest      = "UpdateMemberRoleRequest"
	SchemaCreateTeamRequest            = "CreateTeamRequest"
	SchemaUpdateTeamRequest            = "UpdateTeamRequest"
	SchemaAddTeamMemberRequest         = "AddTeamMemberRequest"
	SchemaUpdateTeamMemberRoleRequest  = "UpdateTeamMemberRoleRequest"

	// Response schemas
	SchemaOrganization     = "Organization"
	SchemaOrganizationList = "OrganizationList"
	SchemaTeam             = "Team"
	SchemaTeamList         = "TeamList"
	SchemaMemberList       = "MemberList"
	SchemaTeamMemberList   = "TeamMemberList"
)
