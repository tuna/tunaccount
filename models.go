package main

const (
	mgoUserColl       = "users"
	mgoPosixGroupColl = "posix_groups"
	mgoFilterTagColl  = "filter_tags"
	mgoCounterColl    = "counters"
)

// keymaps
var userldap2bson = map[string]string{
	"uidNumber":  "_id",
	"gidNumber":  "gid",
	"mail":       "email",
	"uid":        "username",
	"cn":         "username",
	"loginShell": "login_shell",
}
var groupldap2bson = map[string]string{
	"gidNumber": "gid",
	"cn":        "name",
	"memberUid": "members",
}
var ldapIntegerFields = map[string]bool{
	"gidNumber": true,
	"uidNumber": true,
}
var ldapListFields = map[string]bool{
	"memberUid": true,
}

// A User is a tuna account
type User struct {
	UID   int    `bson:"_id" json:"uid" ldap:"uidNumber"`
	GID   int    `bson:"gid" json:"gid" ldap:"gidNumber"`
	Name  string `bson:"name" json:"name"`
	Email string `bson:"email" json:"email" ldap:"mail"`
	Phone string `bson:"phone" json:"phone"`

	Username   string `bson:"username" json:"username" ldap:"uid,cn"`
	Password   string `bson:"password" json:"password" ldap:"userPassword"`
	LoginShell string `bson:"login_shell" json:"login_shell" ldap:"loginShell"`

	IsActive bool `bson:"is_active" json:"is_active"`
	IsAdmin  bool `bson:"is_admin" json:"is_admin"`

	SSHKeys []string `bson:"ssh_keys"`

	Tags []string `bson:"tags" json:"tags"`
}

// Authenticate user with passwd
func (u *User) Authenticate(password string) bool {
	return validateSSHA(password, u.Password)
}

// Passwd set user's password
func (u *User) Passwd(password string) *User {
	u.Password = generateSSHA(password)
	return u
}

// A PosixGroup maps to a posix user group
type PosixGroup struct {
	GID      int      `bson:"gid" json:"gid" ldap:"gidNumber"`
	Name     string   `bson:"name" json:"name" ldap:"cn"`
	Tag      string   `bson:"tag" json:"tag"`
	IsActive bool     `bson:"is_active" json:"is_active"`
	Members  []string `bson:"members" json:"members" ldap:"memberUid"`
}

// A FilterTag can be used to filter users and groups
// a typical filterTag is the server hostname
type FilterTag struct {
	Name string `bson:"_id" json:"name"`
	Desc string `bson:"desc" json:"desc"`
}

type mongoCounter struct {
	ID  string `bson:"_id"`
	Seq int    `bson:"seq"`
}

// A DBDump contains data exported by or
// can be imported to tunaccount
type DBDump struct {
	Users       []User       `json:"users"`
	PosixGroups []PosixGroup `json:"posix_groups"`
}
