use chrono::{DateTime, NaiveDate, NaiveTime, Utc};
use rust_decimal::Decimal;
use serde::{Deserialize, Serialize};
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserBase {
    pub age: Option<i64>,
    pub avatar: Option<String>,
    pub balance: Decimal,
    pub bio: Option<String>,
    pub birthdate: Option<NaiveDate>,
    pub external_id: Option<Uuid>,
    pub is_active: bool,
    pub last_seen: Option<DateTime<Utc>>,
    pub login_time: Option<NaiveTime>,
    pub role: String,
    pub score: Option<f64>,
    pub settings: Option<serde_json::Value>,
    pub username: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct User {
    pub id: Uuid,
    pub age: Option<i64>,
    pub avatar: Option<String>,
    pub balance: Decimal,
    pub bio: Option<String>,
    pub birthdate: Option<NaiveDate>,
    pub external_id: Option<Uuid>,
    pub is_active: bool,
    pub last_seen: Option<DateTime<Utc>>,
    pub login_time: Option<NaiveTime>,
    pub role: String,
    pub score: Option<f64>,
    pub settings: Option<serde_json::Value>,
    pub username: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}
