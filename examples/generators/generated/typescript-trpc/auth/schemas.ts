import { z } from "zod";

// user schemas
const userBaseSchema = z.object({
  age: z.number().int().nullable(),
  avatar: z.string().nullable(),
  balance: z.number().default(0),
  bio: z.string().nullable(),
  birthdate: z.date().nullable(),
  external_id: z.string().uuid().nullable(),
  is_active: z.boolean().default(true),
  last_seen: z.date().nullable(),
  login_time: z.string().nullable(),
  role: z.enum(["admin", "editor", "viewer"]),
  score: z.number().nullable(),
  settings: z.record(z.any()).nullable(),
  username: z.string(),
});

const userSchema = userBaseSchema.extend({
  id: z.string().uuid(),
  created_at: z.date(),
  updated_at: z.date(),
});

type UserBase = z.infer<typeof userBaseSchema>;
type User = z.infer<typeof userSchema>;
