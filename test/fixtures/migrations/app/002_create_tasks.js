migration(m => {
	m.create_table("app.task", t => {
		t.id()
		t.uuid("user_id")
		t.string("title", 200)
		t.text("description").optional()
		t.string("status", 20).default("pending")
		t.datetime("due_date").optional()
		t.timestamps()
	})
	m.create_index("app.task", ["user_id"], { name: "idx_app_task_user_id" })
	m.create_index("app.task", ["status"], { name: "idx_app_task_status" })
})
