migration(m => {
	m.create_table("catalog.category", t => {
		t.id()
		t.string("name", 100)
		t.string("slug", 100).unique()
		t.uuid("parent_id").optional()
		t.integer("sort_order").default(0)
		t.boolean("is_active").default(true)
		t.timestamps()
	})

	m.create_table("catalog.product", t => {
		t.id()
		t.string("sku", 50).unique()
		t.string("name", 200)
		t.string("slug", 200).unique()
		t.text("description").optional()
		t.decimal("price", 10, 2)
		t.decimal("compare_at_price", 10, 2).optional()
		t.integer("stock_quantity").default(0)
		t.boolean("is_active").default(true)
		t.boolean("is_featured").default(false)
		t.timestamps()
	})

	m.create_table("catalog.product_category", t => {
		t.id()
		t.uuid("product_id")
		t.uuid("category_id")
	})

	m.create_index("catalog.product_category", ["product_id", "category_id"], { unique: true, name: "idx_catalog_product_category_prod_cat" })
})
