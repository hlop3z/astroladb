migration(m => {
	m.create_table("auth.role", t => {
		t.id()
		t.string("name", 50).unique()
		t.text("description").optional()
		t.timestamps()
	})

	m.create_table("auth.permission", t => {
		t.id()
		t.string("name", 100).unique()
		t.string("resource", 100)
		t.string("action", 50)
		t.timestamps()
	})

	m.create_table("auth.role_permission", t => {
		t.id()
		t.uuid("role_id")
		t.uuid("permission_id")
		t.timestamps()
	})

	m.create_index("auth.role_permission", ["role_id", "permission_id"], { unique: true, name: "idx_auth_role_permission_role_permission" })
})
