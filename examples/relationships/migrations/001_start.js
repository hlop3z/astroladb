// Migration: start

export default migration({
  up(m) {
    // Table: auth.role
    m.create_table("auth.role", (t) => {
      t.id();
      t.string("name", 100).unique();
      t.timestamps();
    });

    // Table: auth.user
    m.create_table("auth.user", (t) => {
      t.id();
      t.string("email", 255).unique();
      t.boolean("is_active").default(true);
      t.string("password", 255);
      t.string("username", 50).unique();
      t.timestamps();
    });

    // Table: auth_role_auth_user
    m.create_table("auth_role_auth_user", (t) => {
      t.belongs_to("auth.role");
      t.belongs_to("auth.user");
    });
    // Index: auth_role_auth_user
    m.create_index("auth_role_auth_user", ["role_id"], {
      name: "idx_auth_role_auth_user_role_id",
    });
    // Index: auth_role_auth_user
    m.create_index("auth_role_auth_user", ["user_id"], {
      name: "idx_auth_role_auth_user_user_id",
    });
    // Index: auth_role_auth_user
    m.create_index("auth_role_auth_user", ["role_id", "user_id"], {
      unique: true,
      name: "idx_auth_role_auth_user_unique",
    });

    // Table: blog.post
    m.create_table("blog.post", (t) => {
      t.id();
      t.belongs_to("auth.user").as("author");
      t.text("body");
      t.datetime("published_at").optional();
      t.string("slug", 255).unique();
      t.enum("status", ["draft", "published", "archived"]).default("draft");
      t.string("title", 200);
      t.datetime("deleted_at").optional();
      t.timestamps();
    });
    // Index: blog.post
    m.create_index("blog.post", ["author_id"]);
  },

  down(m) {
    m.drop_table("blog.post");
    m.drop_table("auth_role_auth_user");
    m.drop_table("auth.user");
    m.drop_table("auth.role");
  },
});
