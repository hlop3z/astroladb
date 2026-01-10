/// <reference types="../../types" />

export default migration(m => {
  m.add_column("auth.users", c =>
    c.string("username", 100).unique().backfill(sql("split_part(email, '@', 1)"))
  )
})
