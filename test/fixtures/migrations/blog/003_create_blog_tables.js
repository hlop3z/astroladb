migration(m => {
	m.create_table("blog.category", t => {
		t.id()
		t.string("name", 100)
		t.string("slug", 100).unique()
		t.text("description").optional()
		t.uuid("parent_id").optional()
		t.timestamps()
	})

	m.create_table("blog.post", t => {
		t.id()
		t.uuid("author_id")
		t.uuid("category_id").optional()
		t.string("title", 200)
		t.string("slug", 255).unique()
		t.text("excerpt").optional()
		t.text("body")
		t.string("status", 20).default("draft")
		t.datetime("published_at").optional()
		t.integer("view_count").default(0)
		t.timestamps()
	})

	m.create_index("blog.post", ["author_id"], { name: "idx_blog_post_author_id" })
	m.create_index("blog.post", ["category_id"], { name: "idx_blog_post_category_id" })
	m.create_index("blog.post", ["status", "published_at"], { name: "idx_blog_post_status_published" })
})
