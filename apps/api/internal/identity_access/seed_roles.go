package identity_access

import "github.com/google/uuid"

// RoleAdminSeedID is the fixed "admin" role from migration 000001_identity_access.up.sql
// and role-builder seeds; used for first-workspace bootstrap bindings.
var RoleAdminSeedID = uuid.MustParse("10000000-0000-0000-0000-000000000001")
