// Auto-generated Rust types from database schema
// Do not edit manually

use chrono::{DateTime, NaiveDate, NaiveTime, Utc};
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum AuthUserRole {
    Admin,
    Editor,
    Viewer,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthUser {
    pub age: Option<i32>,
    pub avatar: Option<Vec<u8>>,
    pub balance: String,
    pub bio: Option<String>,
    pub birthdate: Option<NaiveDate>,
    pub external_id: Option<String>,
    pub id: String,
    pub is_active: bool,
    pub last_seen: Option<DateTime<Utc>>,
    pub login_time: Option<NaiveTime>,
    pub role: AuthUserRole,
    pub score: Option<f32>,
    pub settings: Option<serde_json::Value>,
    pub username: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// Schema URI to type name mapping.
pub const TYPES: &[(&str, &str)] = &[("auth.user", "AuthUser")];
