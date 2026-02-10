migration(m => {
	m.create_table("blog.tag", t => {
		t.id()
		t.string("name", 50).unique()
		t.string("slug", 50).unique()
		t.timestamps()
	})

	m.create_table("blog.post_tag", t => {
		t.id()
		t.uuid("post_id")
		t.uuid("tag_id")
		t.timestamps()
	})

	m.create_index("blog.post_tag", ["post_id", "tag_id"], { unique: true, name: "idx_blog_post_tag_post_tag" })
	m.create_index("blog.post_tag", ["tag_id"], { name: "idx_blog_post_tag_tag_id" })
})
