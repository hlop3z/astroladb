migration(m => {
	m.add_column("app.task", c => c.integer("priority").default(0))
})
