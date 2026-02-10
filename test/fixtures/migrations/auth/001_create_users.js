migration(m => {
	m.create_table("auth.user", t => {
		t.id()
		t.string("email", 255).unique()
		t.string("password_hash", 255)
		t.boolean("is_active").default(true)
		t.boolean("is_verified").default(false)
		t.datetime("last_login").optional()
		t.timestamps()
	})
})
