package organizations

// Schema names for OpenAPI specification generation.
//
// These constants define the OpenAPI schema names for organizations request/response types.
// They are used in route metadata to generate accurate API documentation with typed
// request/response examples.
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
	SchemaMember           = "Member"
	SchemaMemberList       = "MemberList"
	SchemaTeamMember       = "TeamMember"
	SchemaTeamMemberList   = "TeamMemberList"
)
