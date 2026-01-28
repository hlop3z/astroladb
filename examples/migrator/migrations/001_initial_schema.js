// Migration: initial_schema
//
// Creates the foundation: auth tables (user, role, profile) with a
// many-to-many junction table, and a blog module (post, comment, tag).

export default migration({
  up(m) {
    // ---------------------------------------------------------------
    // Auth Module
    // ---------------------------------------------------------------

    // Table: auth.role
    m.create_table("auth.role", (t) => {
      t.id();
      t.string("name", 100).unique();
      t.text("description").optional();
      t.timestamps();
    });

    // Table: auth.user
    m.create_table("auth.user", (t) => {
      t.id();
      t.string("email", 255).unique();
      t.string("username", 50).unique();
      t.string("password", 255);
      t.boolean("is_active").default(true);
      t.timestamps();
    });

    // Table: auth.profile (one-to-one with auth.user)
    m.create_table("auth.profile", (t) => {
      t.id();
      t.one_to_one("auth.user");
      t.string("display_name", 100).optional();
      t.text("bio").optional();
      t.string("avatar_url", 500).optional();
      t.date("birth_date").optional();
      t.timestamps();
    });

    // Junction: auth_role ↔ auth_user (many-to-many)
    m.create_table("auth_role_auth_user", (t) => {
      t.belongs_to("auth.role");
      t.belongs_to("auth.user");
    });
    m.create_index("auth_role_auth_user", ["role_id"], {
      name: "idx_arau_role",
    });
    m.create_index("auth_role_auth_user", ["user_id"], {
      name: "idx_arau_user",
    });
    m.create_index("auth_role_auth_user", ["role_id", "user_id"], {
      unique: true,
      name: "idx_arau_unique",
    });

    // ---------------------------------------------------------------
    // Blog Module
    // ---------------------------------------------------------------

    // Table: blog.post
    m.create_table("blog.post", (t) => {
      t.id();
      t.belongs_to("auth.user").as("author");
      t.string("title", 200);
      t.string("slug", 255).unique();
      t.text("body");
      t.text("excerpt").optional();
      t.enum("status", ["draft", "published", "archived"]).default("draft");
      t.datetime("published_at").optional();
      t.json("metadata").optional();
      t.soft_delete();
      t.timestamps();
    });
    m.create_index("blog.post", ["author_id"], {
      name: "idx_post_author",
    });
    m.create_index("blog.post", ["status"], {
      name: "idx_post_status",
    });

    // Table: blog.comment
    m.create_table("blog.comment", (t) => {
      t.id();
      t.belongs_to("blog.post");
      t.belongs_to("auth.user").as("commenter");
      t.text("body");
      t.boolean("is_approved").default(false);
      t.soft_delete();
      t.timestamps();
    });
    m.create_index("blog.comment", ["post_id"], {
      name: "idx_comment_post",
    });
    m.create_index("blog.comment", ["commenter_id"], {
      name: "idx_comment_commenter",
    });

    // Table: blog.tag
    m.create_table("blog.tag", (t) => {
      t.id();
      t.string("name", 50).unique();
      t.string("slug", 60).unique();
      t.sortable();
      t.timestamps();
    });

    // Junction: blog_post ↔ blog_tag (many-to-many)
    m.create_table("blog_post_blog_tag", (t) => {
      t.belongs_to("blog.post");
      t.belongs_to("blog.tag");
    });
    m.create_index("blog_post_blog_tag", ["post_id", "tag_id"], {
      unique: true,
      name: "idx_bpbt_unique",
    });
  },

  down(m) {
    m.drop_table("blog_post_blog_tag");
    m.drop_table("blog.tag");
    m.drop_table("blog.comment");
    m.drop_table("blog.post");
    m.drop_table("auth_role_auth_user");
    m.drop_table("auth.profile");
    m.drop_table("auth.user");
    m.drop_table("auth.role");
  },
});
