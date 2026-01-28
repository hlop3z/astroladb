// Auto-generated TypeScript types from database schema
// Do not edit manually

export interface AuthUser {
  age?: number;
  avatar?: string;
  balance: string;
  bio?: string;
  birthdate?: string;
  external_id?: string;
  id: string;
  is_active: boolean;
  last_seen?: string;
  login_time?: string;
  role: "admin" | "editor" | "viewer";
  score?: number;
  settings?: Record<string, unknown>;
  username: string;
  created_at: string;
  updated_at: string;
}

// Schema URI to type name mapping
export const TYPES: Record<string, string> = {
  "auth.user": "AuthUser",
};
