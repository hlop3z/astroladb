migration(m => {
	m.create_table("app.user", t => {
		t.id()
		t.string("email", 255).unique()
		t.string("name", 100)
		t.timestamps()
	})
})
