package acct

const (
	read  int = iota + 1
	Write
)

const (
	TableNameMeal = "meals"
	TableNameUser = "users"
)

// Permission represents the authorized access to a specific component of API functionality.
type Permission struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	TableName string
	level     int
	IsAll     bool
}

// Permissions is a slice of permissions with added functionality.
type Permissions []Permission

// Has returns whether or not the permissions has the given permission.
func (perms Permissions) Has(p Permission) bool {
	for _, pr := range perms {
		if pr == p {
			return true
		}
	}
	return false
}

var (
	// NoPerm represents the lack of a permission
	NoPerm        = Permission{ID: 0, Name: ""}
	ReadUsers     = Permission{ID: 1, Name: "read:users", TableName: TableNameUser, level: read}
	ReadAllUsers  = Permission{ID: 2, Name: "read:all_users", TableName: TableNameUser, level: read, IsAll: true}
	WriteUsers    = Permission{ID: 3, Name: "write:users", TableName: TableNameUser, level: Write}
	WriteAllUsers = Permission{ID: 4, Name: "write:all_users", TableName: TableNameUser, level: Write, IsAll: true}
	ReadMeals     = Permission{ID: 5, Name: "read:meals", TableName: TableNameMeal, level: read}
	ReadAllMeals  = Permission{ID: 6, Name: "read:all_meals", TableName: TableNameMeal, level: read, IsAll: true}
	WriteMeals    = Permission{ID: 7, Name: "write:meals", TableName: TableNameMeal, level: Write}
	WriteAllMeals = Permission{ID: 8, Name: "write:all_meals", TableName: TableNameMeal, level: Write, IsAll: true}
)

// AllPermissions returns a list of all permissions.
func AllPermissions() []Permission {
	return []Permission{ReadUsers, WriteUsers, ReadMeals, WriteMeals}
}

// Group represents a named grouping of permissions for ease of assignment.
type Group struct {
	ID          int64        `json:"id"`
	Name        string       `json:"name"`
	Permissions []Permission `json:"permissions"`
}

var (
	// Admins can manage users and generally do everything
	Admin = Group{ID: 1, Name: "Admin", Permissions: []Permission{
		ReadAllUsers,
		WriteAllUsers,
		ReadAllMeals,
		WriteAllMeals,
	}}
	// UserManager is a group allowed to CRUD users
	UserManager = Group{ID: 2, Name: "UserManager", Permissions: []Permission{
		ReadAllUsers,
		WriteAllUsers,
		ReadMeals,
		WriteMeals,
	}}
	// Regular is a group allowed to only view data
	Regular = Group{ID: 3, Name: "Regular", Permissions: []Permission{
		ReadUsers,
		ReadMeals,
		WriteMeals,
	}}
)

// Groups returns a list of all groups.
func Groups() []Group {
	return []Group{
		Admin,
		UserManager,
		Regular,
	}
}

// GroupForID returns the group for the given id.
// If no group is found, nil is returned.
func GroupForID(id int64) *Group {
	for _, p := range Groups() {
		if p.ID == id {
			return &p
		}
	}
	return nil
}

func (group *Group) GetPermission(tableName string, level int) Permission {
	for _, perm := range group.Permissions {
		if perm.TableName == tableName && perm.level == level {
			return perm
		}
	}
	return NoPerm
}
