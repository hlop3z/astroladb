migration(m => {
	m.create_table("inventory.warehouse", t => {
		t.id()
		t.string("name", 100)
		t.string("code", 20).unique()
		t.boolean("is_active").default(true)
		t.timestamps()
	})

	m.create_table("inventory.stock", t => {
		t.id()
		t.uuid("product_id")
		t.uuid("warehouse_id")
		t.integer("quantity").default(0)
		t.integer("reserved").default(0)
		t.timestamps()
	})

	m.create_index("inventory.stock", ["product_id", "warehouse_id"], { unique: true, name: "idx_inventory_stock_product_warehouse" })
})
