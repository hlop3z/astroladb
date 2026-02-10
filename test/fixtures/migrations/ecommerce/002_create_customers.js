migration(m => {
	m.create_table("customer.account", t => {
		t.id()
		t.string("email", 255).unique()
		t.string("password_hash", 255)
		t.string("first_name", 50)
		t.string("last_name", 50)
		t.string("phone", 20).optional()
		t.boolean("is_active").default(true)
		t.boolean("email_verified").default(false)
		t.timestamps()
	})

	m.create_table("customer.address", t => {
		t.id()
		t.uuid("customer_id")
		t.string("label", 50).optional()
		t.string("street_1", 200)
		t.string("street_2", 200).optional()
		t.string("city", 100)
		t.string("state", 100)
		t.string("postal_code", 20)
		t.string("country", 2)
		t.boolean("is_default").default(false)
		t.timestamps()
	})

	m.create_index("customer.address", ["customer_id"], { name: "idx_customer_address_customer_id" })
})
