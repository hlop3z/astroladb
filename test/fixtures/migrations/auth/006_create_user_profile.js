migration(m => {
	m.create_table("auth.profile", t => {
		t.id()
		t.uuid("user_id").unique()
		t.string("display_name", 100).optional()
		t.text("bio").optional()
		t.string("avatar_url", 500).optional()
		t.string("website", 500).optional()
		t.string("location", 100).optional()
		t.timestamps()
	})

	m.add_column("auth.user", c => c.uuid("role_id").optional())
	m.create_index("auth.user", ["role_id"])  // Name auto-generated: idx_auth_user_role_id
})
