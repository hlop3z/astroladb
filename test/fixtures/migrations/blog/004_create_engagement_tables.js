migration(m => {
	m.create_table("blog.comment", t => {
		t.id()
		t.uuid("post_id")
		t.uuid("author_id").optional()
		t.uuid("parent_id").optional()
		t.text("content")
		t.boolean("is_approved").default(false)
		t.timestamps()
	})

	m.create_table("blog.reaction", t => {
		t.id()
		t.uuid("post_id")
		t.uuid("user_id")
		t.string("emoji", 10)
		t.timestamps()
	})

	m.create_index("blog.comment", ["post_id"], { name: "idx_blog_comment_post_id" })
	m.create_index("blog.comment", ["author_id"], { name: "idx_blog_comment_author_id" })
	m.create_index("blog.reaction", ["post_id", "user_id"], { unique: true, name: "idx_blog_reaction_post_user" })
})
